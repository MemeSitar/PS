package main

import "fmt"

func printFlags() {
	fmt.Println("Logging level of:", *loglevelPtr)
	fmt.Println("Polling of controller:", *pollingPtr)
	fmt.Println("Max number of workers:", *maxWorkersPtr)
	fmt.Println("Number of map shards:", *shardNumPtr)
	fmt.Println("Producer with delay of:", *tPtr*100)
}
