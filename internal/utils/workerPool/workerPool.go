package workerPool

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"

	"wfts/internal/model"
	"wfts/pkg/logger"
)

type WorkerPool struct {
	log 		*logger.Logger
	buf 		chan struct{}
	quit      	chan struct{}
	crawlHeap 	CrawlStream
	wg        	*sync.WaitGroup
	ctx 		context.Context
	workers   	int32
}

func NewWorkerPool(size int, queueCapacity int, c context.Context, l *logger.Logger) *WorkerPool {
	qheap := make(CrawlStream, 0)
	heap.Init(&qheap)
	wp := &WorkerPool{
		log:       	l,
		buf: 		make(chan struct{}, queueCapacity),
		quit:      	make(chan struct{}),
		crawlHeap:  qheap,
		wg:        	new(sync.WaitGroup),
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

	select{
	case wp.buf <- struct{}{}:
		wp.crawlHeap.Push(task)
	default:
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
			if f := wp.crawlHeap.Pop().(model.CrawlNode); f.Activation != nil {
				f.Activation()
			}
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