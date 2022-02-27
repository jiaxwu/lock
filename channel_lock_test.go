package lock

import (
	"sync"
	"testing"
)

func TestNewChannelLock(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)
	count := 0
	times := 100000
	l := NewChannelLock()
	go func() {
		defer wg.Done()
		for i := 0; i < times; i++ {
			l.Lock()
			count++
			l.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < times; i++ {
			l.Lock()
			count++
			l.Unlock()
		}
	}()
	wg.Wait()
	if count != times*2 {
		t.Errorf("count = %d", count)
	}
}

func TestChannelTryLock(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)
	count := 0
	times := 100000
	l := NewChannelLock()
	go func() {
		defer wg.Done()
		for i := 0; i < times; i++ {
			for !l.TryLock() {
			}
			count++
			l.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < times; i++ {
			for !l.TryLock() {
			}
			count++
			l.Unlock()
		}
	}()
	wg.Wait()
	if count != times*2 {
		t.Errorf("count = %d", count)
	}
}
