package lock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"sync"
	"testing"
	"time"
)

func TestNewLock(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	locker := NewLocker(client, ttl, tryLockInterval)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		l := locker.GetLock("test")
		err := l.TryLock(context.Background())
		if err != nil {
			t.Error(err)
		}
		time.Sleep(ttl)
		err = l.Unlock(context.Background())
		if err != nil {
			t.Error(err)
		}
	}()
	time.Sleep(time.Second)
	go func() {
		defer wg.Done()
		l := locker.GetLock("test")
		err := l.TryLock(context.Background())
		if err != nil && !errors.Is(err, ErrLockFailed) {
			t.Error(err)
		}
	}()
	wg.Wait()
}

func TestNewLock2(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	locker := NewLocker(client, ttl, tryLockInterval)
	var wg sync.WaitGroup
	wg.Add(2)
	count := 0
	times := 100000
	go func() {
		defer wg.Done()
		for i := 0; i < times; i++ {
			l := locker.GetLock("test")
			err := l.Lock(context.Background())
			if err != nil {
				t.Error(err)
			}
			count++
			err = l.Unlock(context.Background())
			if err != nil {
				t.Error(err)
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < times; i++ {
			l := locker.GetLock("test")
			err := l.Lock(context.Background())
			if err != nil {
				t.Error(err)
			}
			count++
			err = l.Unlock(context.Background())
			if err != nil {
				t.Error(err)
			}
		}
	}()
	wg.Wait()
	if count != times*2 {
		t.Errorf("count = %d", count)
	}
}
