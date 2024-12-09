package main

import "fmt"

func printFlags() {
	fmt.Println("Logging level of:", *loglevelPtr)
	fmt.Println("Polling of controller:", *pollingPtr)
	fmt.Println("Max number of workers:", *maxWorkersPtr)
	fmt.Println("Number of map shards:", *shardNumPtr)
	fmt.Println("Producer with delay of:", *tPtr*100)
}

func printResults() {
	for i, arr := range slovar.arr {
		slovar.lock[i].Lock()
		fmt.Println(arr)
		slovar.lock[i].Unlock()
	}
}

func query(word string) []uint64 {
	var indices []uint64
	for i, arr := range slovar.arr {
		slovar.lock[i].Lock()
		indices = append(indices, arr[word]...)
		slovar.lock[i].Unlock()
	}
	return indices
}
