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

type Entry struct {
	Id   uint64
	Word string
}

// colors
var Reset = "\033[0m"
var Red = "\033[31m"

var loglevelPtr = flag.Int("l", 0, "set log level")
var pollingPtr = flag.Int("poll", 20, "set polling time of controller")
var wg sync.WaitGroup
var re = regexp.MustCompile(`\b[a-zA-Z0-9]{4,}\b`)
var w = log.New(os.Stderr, "WORKER: ", 0)
var c = log.New(os.Stderr, "CONTROLLER: ", 0)
var lock sync.Mutex
var slovar = make(map[string][]uint64)

func worker(id int, kanal chan socialNetwork.Task) {
	var current socialNetwork.Task
	var todo []Entry
	for {
		current = <-kanal
		if *loglevelPtr == 1 {
			w.Println(id, "processing task", current.Id)
		}

		// spucaj stringe
		words := re.FindAllString(current.Data, -1)
		// čez vse besede tolower in dodaj v slovar
		for _, word := range words {
			word = strings.ToLower(word)
			todo = append(todo, Entry{current.Id, word})
		}
		if lock.TryLock() {
			if *loglevelPtr == 2 {
				w.Println(id, "successfully locked, copying:", len(todo))
			}
			for len(todo) > 0 {
				popped := todo[len(todo)-1]
				slovar[popped.Word] = append(slovar[popped.Word], popped.Id)
				todo = todo[:len(todo)-1]
			}

			lock.Unlock()
		}
	}
}

func controller(kanal chan socialNetwork.Task) {
	/*wnum := 100
	fmt.Println("Starting", wnum, "workers.")
	for i := 0; i < wnum; i++ {
		go worker(i, kanal)
	}*/
	//defer wg.Done()
	/*for {
		fmt.Println(<-kanal)
	}*/
	prev := 0
	diff := 0
	if *loglevelPtr == 3 {
		for {
			time.Sleep(time.Duration(*pollingPtr) * time.Millisecond)
			tmp := len(kanal)
			diff = tmp - prev
			if tmp > 9500 {
				c.Printf("%scurrent diff: %5d current len: %d%s\n", Red, diff, tmp, Reset)
			} else {
				c.Printf("current diff: %5d current len: %d\n", diff, tmp)
			}
			//c.Println("current diff:", diff, "\tcurrent len:", tmp)
			prev = tmp
		}
	}
}

func main() {
	var wnumPtr = flag.Int("w", 10, "set log level")
	var tPtr = flag.Int("t", 10, "delay between tasks")
	flag.Parse()

	// Definiramo nov generator
	var producer socialNetwork.Q
	// Inicializiramo generator. Parameter določa zakasnitev med zahtevki
	fmt.Println("Producer with delay of", *tPtr*100)
	producer.New(*tPtr * 100)

	start := time.Now()
	// Delavec, samo prevzema zahtevke
	/*go func() {
		for {
			<-producer.TaskChan
		}
	}()*/
	go controller(producer.TaskChan)

	fmt.Println("Starting", *wnumPtr, "workers.")
	for i := 0; i < *wnumPtr; i++ {
		go worker(i, producer.TaskChan)
	}
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
