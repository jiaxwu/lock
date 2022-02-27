package lock

import (
	"log"
)

type ChannelLock struct {
	c chan struct{}
}

func NewChannelLock() *ChannelLock {
	return &ChannelLock{
		c: make(chan struct{}, 1),
	}
}

func (l *ChannelLock) TryLock() bool {
	select {
	case l.c <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *ChannelLock) Lock() {
	l.c <- struct{}{}
}

func (l *ChannelLock) Unlock() {
	select {
	case <-l.c:
	default:
		log.Fatal("sync: unlock of unlocked mutex")
	}
}
