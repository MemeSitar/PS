package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"q-index/socialNetwork"
	"regexp"
	"strings"
	"sync"
	"time"
)

// colors
var RESET = "\033[0m"
var RED = "\033[31m"

// global flags
var loglevelPtr = flag.Int("l", 0, "set log level")
var pollingPtr = flag.Int("poll", 20, "set polling time of controller")
var maxWorkersPtr = flag.Int("wmax", 80, "max number of workers")
var shardNumPtr = flag.Int("shards", 1, "number of shards")
var tPtr = flag.Int("t", 10, "delay between tasks")

var wg sync.WaitGroup
var re = regexp.MustCompile(`\b[a-zA-Z0-9]{4,}\b`)
var w = log.New(os.Stderr, "WORKER: ", 0)
var c = log.New(os.Stderr, "CONTROLLER: ", 0)
var slovar Slovar

type Slovar struct {
	arr  []map[string][]uint64
	lock []sync.Mutex
}

type Entry struct {
	Id   uint64
	Word string
}

func worker(id int, kanal chan socialNetwork.Task, stop chan int) {
	defer wg.Done()
	var current socialNetwork.Task
	todo := make([][]Entry, *shardNumPtr)
	for {
		select {
		case <-stop:
			if *loglevelPtr > 3 {
				w.Println(id, "got stop signal, clearing task list.")
			}
			for x, arr := range todo {
				if len(arr) > 0 {
					slovar.lock[x].Lock()
					if *loglevelPtr > 3 {
						w.Println(id, "successfully locked shard", x, "copying:", len(arr))
					}
					for len(arr) > 0 {
						popped := arr[len(arr)-1]
						slovar.arr[x][popped.Word] = append(slovar.arr[x][popped.Word], popped.Id)
						arr = arr[:len(arr)-1]
					}
					if *loglevelPtr > 3 {
						w.Println(id, "unlocking shard", x, "current:", len(arr))
					}
					slovar.lock[x].Unlock()
					todo[x] = make([]Entry, 0)
				}
			}
			return
		default:

		}
		current = <-kanal
		if *loglevelPtr > 5 {
			w.Println(id, "processing task", current.Id)
		}

		// spucaj stringe
		words := re.FindAllString(current.Data, -1)
		// čez vse besede tolower in dodaj v slovar
		for _, word := range words {
			word = strings.ToLower(word)
			todo[current.Id%uint64(*shardNumPtr)] = append(todo[current.Id%uint64(*shardNumPtr)], Entry{current.Id, word})
		}
		for x, arr := range todo {
			if len(arr) > 0 && slovar.lock[x].TryLock() {
				if *loglevelPtr > 4 {
					w.Println(id, "successfully locked shard", x, "copying:", len(arr))
				}
				for len(arr) > 0 {
					popped := arr[len(arr)-1]
					slovar.arr[x][popped.Word] = append(slovar.arr[x][popped.Word], popped.Id)
					arr = arr[:len(arr)-1]
				}
				slovar.lock[x].Unlock()
				todo[x] = make([]Entry, 0)
			} else if *loglevelPtr > 4 && len(arr) > 0 {
				w.Println(id, "couldn't lock shard", x, "keeping:", len(arr))
			}
		}
	}
}

func controller(kanal chan socialNetwork.Task, quit chan int) {
	wg.Add(1)
	defer wg.Done()
	timer := 0
	var stop = make(chan int)
	var WP = WorkerPool{0, 0, *maxWorkersPtr, stop, kanal}
	WP.addWorker(1)

	// start the main loop
	prev := 0
	diff := 0
	for {
		time.Sleep(time.Duration(*pollingPtr) * time.Millisecond)
		tmp := len(kanal)
		diff = tmp - prev
		prev = tmp

		// test if we are exiting program
		select {
		case <-quit:
			if *loglevelPtr > 0 {
				c.Println("received quit signal, stopping", WP.WorkerNum, "workers.")
			}
			WP.stopAll()
			return
		default:
			// debug prints to see the performance
			if *loglevelPtr > 0 {
				if tmp > 9500 {
					c.Printf("%scurrent diff: %5d current len: %d%s\n", RED, diff, tmp, RESET)
				} else {
					c.Printf("current diff: %5d current len: %d\n", diff, tmp)
				}
			}
			if timer > 0 {
				timer--
			}
		}

		// dinamično dodajanje/odstranjevanje
		if tmp > 9500 {
			if *loglevelPtr > 1 {
				c.Println("adding", 10, "workers. Current workers:", WP.WorkerNum)
			}
			timer = 5
			WP.addWorker(10)
		} else if diff > 100 && timer == 0 {
			if *loglevelPtr > 1 {
				c.Println("adding", diff/100, "workers. Current workers:", WP.WorkerNum)
			}
			timer = 5
			WP.addWorker(diff / 100)
		} else if diff < -500 && timer == 0 {
			if *loglevelPtr > 1 {
				c.Println("removing", diff/500*-1, "workers. Current workers:", WP.WorkerNum)
			}
			timer = 5
			WP.removeWorker(diff / 500 * -1)
		}

	}
}

func main() {
	flag.Parse()
	printFlags()
	slovar = Slovar{make([]map[string][]uint64, *shardNumPtr), make([]sync.Mutex, *shardNumPtr)}
	slovar.arr = make([]map[string][]uint64, *shardNumPtr)
	for i := 0; i < *shardNumPtr; i++ {
		slovar.arr[i] = make(map[string][]uint64)
	}
	slovar.lock = make([]sync.Mutex, *shardNumPtr)
	// Definiramo nov generator
	var producer socialNetwork.Q
	// Inicializiramo generator. Parameter določa zakasnitev med zahtevki

	producer.New(*tPtr * 100)

	var stopController = make(chan int)
	go controller(producer.TaskChan, stopController)

	start := time.Now()
	// Zaženemo generator
	go producer.Run()
	// Počakamo 5 sekund
	time.Sleep(time.Second * 5)
	// Ustavimo generator
	producer.Stop()
	// Počakamo, da se vrsta sprazni
	for !producer.QueueEmpty() {
	}
	elapsed := time.Since(start)
	fmt.Println("STOPPING CONTROLLER")
	stopController <- 1
	wg.Wait()

	// Izpišemo število generiranih zahtevkov na sekundo
	fmt.Printf("Processing rate: %f MReqs/s\n", float64(producer.N)/float64(elapsed.Seconds())/1000000.0)
	// Izpišemo povprečno dolžino vrste v čakalnici
	fmt.Printf("Average queue length: %.2f %%\n", producer.GetAverageQueueLength())
	// Izpišemo največjo dolžino vrste v čakalnici
	fmt.Printf("Max queue length %.2f %%\n", producer.GetMaxQueueLength())
	for _, arr := range slovar.arr {
		fmt.Println(arr)
	}
}

/* LOG LEVELS:
# log level 0
Only print main function. Starting parameters and final results.

# log level 1
Output state of queue. Simple evaluation of how it's changing. RED if bad.

# log level 2
Output dynamic management of controller. How it is adding and removing workers.

# log level 3
Output basic worker state, locking.

# log level 4
Output worker's tasks.

*/
