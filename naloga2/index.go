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
var Reset = "\033[0m"
var Red = "\033[31m"

// global flags
var loglevelPtr = flag.Int("l", 0, "set log level")
var pollingPtr = flag.Int("poll", 20, "set polling time of controller")
var maxWorkersPtr = flag.Int("wmax", 80, "max number of workers")

var wg sync.WaitGroup
var re = regexp.MustCompile(`\b[a-zA-Z0-9]{4,}\b`)
var w = log.New(os.Stderr, "WORKER: ", 0)
var c = log.New(os.Stderr, "CONTROLLER: ", 0)

var lock sync.Mutex
var slovar = make(map[string][]uint64)

type WorkerPool struct {
	Uid       int
	WorkerNum int
	WorkerMax int
	StopChan  chan int
	TaskChan  chan socialNetwork.Task
}

type Entry struct {
	Id   uint64
	Word string
}

func (WP *WorkerPool) addWorker(num int) int {
	for i := 0; i < num && WP.WorkerNum < WP.WorkerMax; i++ {
		WP.Uid++
		wg.Add(1)
		go worker(WP.Uid, WP.TaskChan, WP.StopChan)
		WP.WorkerNum++
	}
	return WP.WorkerNum
}

func (WP *WorkerPool) removeWorker(num int) int {
	for i := 0; i < num && WP.WorkerNum > 1; i++ {
		WP.StopChan <- 1
		WP.WorkerNum--
	}
	return WP.WorkerNum
}

func (WP *WorkerPool) stopAll() {
	for ; WP.WorkerNum > 0; WP.WorkerNum-- {
		WP.StopChan <- 1
	}
}

func worker(id int, kanal chan socialNetwork.Task, stop chan int) {
	defer wg.Done()
	var current socialNetwork.Task
	for {
		todo := make([]Entry, 0)
		current = <-kanal
		if *loglevelPtr > 5 {
			w.Println(id, "processing task", current.Id)
		}

		// spucaj stringe
		words := re.FindAllString(current.Data, -1)
		// čez vse besede tolower in dodaj v slovar
		for _, word := range words {
			word = strings.ToLower(word)
			todo = append(todo, Entry{current.Id, word})
		}
		if len(words) > 0 && lock.TryLock() {
			if *loglevelPtr > 5 {
				w.Println(id, "successfully locked, copying:", len(todo))
			}
			for len(todo) > 0 {
				popped := todo[len(todo)-1]
				slovar[popped.Word] = append(slovar[popped.Word], popped.Id)
				todo = todo[:len(todo)-1]
			}

			lock.Unlock()
		}
		select {
		case <-stop:
			if len(todo) > 0 {

				if *loglevelPtr > 3 {
					w.Println(id, "got stop signal, clearing task list.")
				}
				lock.Lock()
				if *loglevelPtr > 3 {
					w.Println(id, "successfully locked, copying:", len(todo))
				}
				for len(todo) > 0 {
					popped := todo[len(todo)-1]
					slovar[popped.Word] = append(slovar[popped.Word], popped.Id)
					todo = todo[:len(todo)-1]
				}

				lock.Unlock()

				if *loglevelPtr > 3 {
					w.Println(id, "task list cleared, terminating.")
				}
				return
			} else {

				if *loglevelPtr > 3 {
					w.Println(id, "Got stop signal, terminating.")
				}
				return
			}
		default:
			continue
		}
	}
}

func controller(kanal chan socialNetwork.Task, quitChan chan int) {
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
		case <-quitChan:
			if *loglevelPtr > 0 {
				c.Println("received quit signal, stopping", WP.WorkerNum, "workers.")
			}
			WP.stopAll()
			return
		default:
			// debug prints to see the performance
			if *loglevelPtr > 0 {
				if tmp > 9500 {
					c.Printf("%scurrent diff: %5d current len: %d%s\n", Red, diff, tmp, Reset)
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
	var tPtr = flag.Int("t", 10, "delay between tasks")
	flag.Parse()
	fmt.Println("Logging level of:", *loglevelPtr)
	fmt.Println("Polling of controller:", *pollingPtr)
	fmt.Println("Max number of workers:", *maxWorkersPtr)
	// Definiramo nov generator
	var producer socialNetwork.Q
	// Inicializiramo generator. Parameter določa zakasnitev med zahtevki
	fmt.Println("Producer with delay of:", *tPtr*100)
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
	//lock.Lock()
	//fmt.Println(slovar)
	//lock.Unlock()
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
