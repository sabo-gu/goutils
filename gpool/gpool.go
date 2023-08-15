package gpool

import "sync"

type GPool struct {
	queue chan int
	wg    *sync.WaitGroup
}

func New(size int) *GPool {
	if size <= 0 {
		size = 1
	}
	return &GPool{
		queue: make(chan int, size),
		wg:    &sync.WaitGroup{},
	}
}

func (p *GPool) Add(delta int) {
	for i := 0; i < delta; i++ {
		p.queue <- 1
	}
	for i := 0; i > delta; i-- {
		<-p.queue
	}
	p.wg.Add(delta)
}

func (p *GPool) Done() {
	<-p.queue
	p.wg.Done()
}

func (p *GPool) Wait() {
	p.wg.Wait()
}
