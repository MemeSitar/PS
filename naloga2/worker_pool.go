package main

import "q-index/socialNetwork"

type WorkerPool struct {
	Uid       int
	WorkerNum int
	WorkerMax int
	StopChan  chan int
	TaskChan  chan socialNetwork.Task
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
