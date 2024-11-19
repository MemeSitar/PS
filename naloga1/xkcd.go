package main

import (
	"container/heap"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/laspp/PS-2024/vaje/naloga-1/koda/xkcd"
)

var wg sync.WaitGroup
var lock sync.Mutex
var mainDict = make(map[string]int)

type Beseda struct {
	key string
	val int
}

type MaxHeap []Beseda

func (h MaxHeap) Len() int           { return len(h) }
func (h MaxHeap) Less(i, j int) bool { return h[i].val > h[j].val }
func (h MaxHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *MaxHeap) Push(x any) {
	*h = append(*h, x.(Beseda))
	// Bubble up to maintain heap property
	i := h.Len() - 1
	for i > 0 {
		parent := (i - 1) / 2
		if (*h)[i].val <= (*h)[parent].val {
			break
		}
		(*h)[i], (*h)[parent] = (*h)[parent], (*h)[i]
		i = parent
	}
}

func (h *MaxHeap) Pop() any {
	old := *h
	n := len(old)
	if n == 0 {
		return nil
	}

	// The last element becomes the new root
	item := old[0]    // This is the root (element to pop)
	*h = old[0 : n-1] // Shrink the heap

	if n == 1 {
		return item // If there's only one element, no need to bubble down
	}

	// Move the last element to the root
	(*h)[0] = old[n-1]

	// Bubble down the new root to restore heap property
	i := 0
	for {
		left := 2*i + 1
		right := 2*i + 2
		largest := i

		// Compare with left child
		if left < n-1 && (*h)[left].val > (*h)[largest].val {
			largest = left
		}

		// Compare with right child
		if right < n-1 && (*h)[right].val > (*h)[largest].val {
			largest = right
		}

		if largest == i {
			break
		}

		// Swap with the larger child
		(*h)[i], (*h)[largest] = (*h)[largest], (*h)[i]
		i = largest
	}

	return item // Return the popped root element
}

func worker(id int, hopsize int, lastid int) {
	defer wg.Done()
	dict := make(map[string]int)
	re := regexp.MustCompile(`\b[a-zA-Z0-9]{4,}\b`)
	//fmt.Println("Running worker", id)
	// glavni loop
	for com := id; com <= lastid; com += hopsize {
		comic, err := xkcd.FetchComic(com)
		nit := ""
		if err == nil {
			//fmt.Println(comic.Title)
			if comic.Transcript == "" {
				nit = comic.Title + " " + comic.Tooltip
			} else {
				nit = comic.Title + " " + comic.Transcript
			}
		} else {
			fmt.Println("Worker", id, "encountered an error")
			return
		}
		words := re.FindAllString(nit, -1)
		for v := range words {
			words[v] = strings.ToLower(words[v])
			dict[words[v]]++
		}
	}
	// vse gre skupaj
	lock.Lock()
	for key, value := range dict {
		mainDict[key] += value
	}
	lock.Unlock()
}

func getLastId() int {
	comic, err := xkcd.FetchComic(0)
	if err == nil {
		return comic.Id
	} else {
		fmt.Println("Iskanje IDjev ni delalo.")
		return 0
	}
}

func main() {
	lastid := getLastId()
	hopsize := 500
	//var slovar = make(map[string]int)
	for i := 0; i < hopsize; i++ {
		wg.Add(1)
		go worker(i, hopsize, lastid)
	}
	wg.Wait()
	fmt.Println("------------------------------------")
	// izpis rezultatov

	h := &MaxHeap{}
	heap.Init(h)
	for key, value := range mainDict {
		//fmt.Println(isMaxHeap(h))
		if value > 100 {

			//fmt.Printf("Word: %s, Count: %d\n", key, value)
		}
		h.Push(Beseda{key, value})
		/*fmt.Println("Heap after pushing:")
		for _, item := range *h {
			fmt.Printf("Word: %s, Count: %d\n", item.key, item.val)
		}*/
	}
	for i := 0; i < 10; i++ {

		beseda := h.Pop().(Beseda)
		fmt.Println(beseda.key, ":", beseda.val)
		//fmt.Println(isMaxHeap(h))
	}
	/*
		maxval := 0
		for key, value := range mainDict {
			if value > maxval {
				maxval = value
				fmt.Printf("'%s': %d\n", key, value)
			}
		}*/
}

func isMaxHeap(h *MaxHeap) bool {
	n := len(*h)
	for i := 0; i < n; i++ {
		left := 2*i + 1
		right := 2*i + 2
		if left < n && (*h)[i].val < (*h)[left].val {
			return false
		}
		if right < n && (*h)[i].val < (*h)[right].val {
			return false
		}
	}
	return true
}
