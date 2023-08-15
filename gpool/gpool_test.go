package gpool

import (
	"log"
	"testing"
	"time"
)

func routine(test int, p *GPool) {
	p.Add(1)
	defer p.Done()
	log.Printf("%d\n", test)
	time.Sleep(1 * time.Second)
	log.Printf("%d succ\n", test)
}

func TestPool(t *testing.T) {
	pool := New(50)
	for i := 0; i < 100; i++ {
		go routine(i, pool)
	}
	pool.Wait()
}
