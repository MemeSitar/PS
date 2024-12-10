package main

import (
	"q-index/socialNetwork"
	"strings"
)

func conditionalWrite(arr []Entry, x int) bool {
	if len(arr) > 0 && slovar.lock[x].TryLock() {

		for len(arr) > 0 {
			popped := arr[len(arr)-1]
			slovar.arr[x][popped.Word] = append(slovar.arr[x][popped.Word], popped.Id)
			arr = arr[:len(arr)-1]
		}
		slovar.lock[x].Unlock()
		return true
	}
	return false
}

func forceWrite(arr []Entry, x int) {
	slovar.lock[x].Lock()

	for len(arr) > 0 {
		popped := arr[len(arr)-1]
		slovar.arr[x][popped.Word] = append(slovar.arr[x][popped.Word], popped.Id)
		arr = arr[:len(arr)-1]
	}

	slovar.lock[x].Unlock()
}

func worker(id int, kanal chan socialNetwork.Task, stop chan int) {
	defer wg.Done()
	var current socialNetwork.Task
	todo := make([][]Entry, *shardNumPtr)
	for {
		// first check if we are stopping
		select {
		case <-stop:
			if *loglevelPtr > 3 {
				w.Println(id, "got stop signal, clearing task list.")
			}
			for x, arr := range todo {
				if len(arr) > 0 {
					forceWrite(arr, x)
					if *loglevelPtr > 3 {
						w.Println(id, "successfully locked shard", x, "copied:", len(arr))
					}
					todo[x] = make([]Entry, 0)
				}
			}
			return
		default:
		}

		// actually process the message
		current = <-kanal
		if *loglevelPtr > 5 {
			w.Println(id, "processing task", current.Id)
		}
		currentShard := current.Id % uint64(*shardNumPtr)
		// spucaj stringe
		words := re.FindAllString(current.Data, -1)
		// čez vse besede tolower in dodaj v slovar
		for _, word := range words {
			word = strings.ToLower(word)
			todo[currentShard] = append(todo[currentShard], Entry{current.Id, word})
		}
		for x, arr := range todo {
			if len(arr) > 0 && conditionalWrite(arr, x) {
				// conditional write successful, clean todo[x] because slices weird
				if *loglevelPtr > 4 {
					w.Println(id, "successfully locked shard", x, "copied:", len(arr))
				}
				todo[x] = make([]Entry, 0)
			}
		}
	}
}
