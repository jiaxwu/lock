package lock

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"testing"
	"time"
)

func TestNewRedLock(t *testing.T) {
	var clients []*redis.Client
	clients = append(clients, redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   0,
	}), redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   1,
	}), redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   2,
	}), redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   3,
	}), redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   4,
	}))
	l := NewRedLock(clients, "test")
	err := l.TryLock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	err = l.TryLock(context.Background())
	if err != nil && !errors.Is(err, ErrLockFailed) {
		t.Fatal(err)
	}
	time.Sleep(ttl)
	err = l.Unlock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
}
