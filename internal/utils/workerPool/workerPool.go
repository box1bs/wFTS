package workerPool

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"

	"wfts/internal/model"
)

type WorkerPool struct {
	buf 		chan struct{}
	quit      	chan struct{}
	crawlHeap 	CrawlStream
	wg        	*sync.WaitGroup
	mu 			*sync.Mutex
	ctx 		context.Context
	workers   	int32
}

func NewWorkerPool(size int, queueCapacity int, c context.Context) *WorkerPool {
	qheap := make(CrawlStream, 0)
	heap.Init(&qheap)
	wp := &WorkerPool{
		buf: 		make(chan struct{}, queueCapacity),
		quit:      	make(chan struct{}),
		crawlHeap:  qheap,
		wg:        	new(sync.WaitGroup),
		mu:			new(sync.Mutex),
		ctx: 		c,
	}
	for range size {
		go wp.worker()
	}
	return wp
}

func (wp *WorkerPool) Submit(task model.CrawlNode) {
	wp.wg.Add(1)
	//wp.log.Write(logger.NewMessage(logger.WORKER_POOL_LAYER, logger.DEBUG, "Submitting task. Buffer: %d, Workers: %d", len(wp.buf), wp.workers))

	orig := task.Activation
	task.Activation = func() {
		defer wp.wg.Done()
		orig()
	}

	wp.mu.Lock()
	select{
	case wp.buf <- struct{}{}:
		wp.crawlHeap.Push(task)
		wp.mu.Unlock()
	default:
		wp.mu.Unlock()
		task.Activation()
	}
}

func (wp *WorkerPool) worker() {
	atomic.AddInt32(&wp.workers, 1)
	defer atomic.AddInt32(&wp.workers, -1)
	for {
		select {
		case _, ok := <-wp.buf:
			if !ok {
				return
			}
			wp.mu.Lock()
			if f := wp.crawlHeap.Pop().(model.CrawlNode); f.Activation != nil {
				wp.mu.Unlock()
				f.Activation()
				continue
			}
			wp.mu.Unlock()
		case <-wp.quit:
			return
		}
	}
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

func (wp *WorkerPool) Stop() {
	close(wp.quit)
	close(wp.buf)
	wp.Wait()
}