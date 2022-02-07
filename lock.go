package lock

import (
	"context"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/go-redis/redis/v8"
	"time"
)

type Lock struct {
	client      *redis.Client // Redis客户端
	script      *redis.Script // 解锁脚本
	resource    string        // 锁定的资源
	randomValue string        // 随机值
	watchDog    chan struct{} // 看门狗
}

func NewLock(client *redis.Client, resource string) *Lock {
	return &Lock{
		client:   client,
		script:   redis.NewScript(unlockScript),
		resource: resource,
	}
}

func (l *Lock) TryLock(ctx context.Context) error {
	randomValue := gofakeit.UUID()
	success, err := l.client.SetNX(ctx, l.resource, randomValue, ttl).Result()
	if err != nil {
		return err
	}
	// 加锁失败
	if !success {
		return ErrLockFailed
	}
	// 加锁成功，启动看门狗
	l.randomValue = randomValue
	go l.startWatchDog()
	return nil
}

func (l *Lock) startWatchDog() {
	l.watchDog = make(chan struct{})
	ticker := time.NewTicker(resetTTLInterval)
	for {
		select {
		case <-ticker.C:
			// 延长锁的过期时间
			timeout, _ := context.WithTimeout(context.Background(), ttl-resetTTLInterval)
			ok, err := l.client.Expire(timeout, l.resource, ttl).Result()
			if err != nil || !ok {
				return
			}
		case <-l.watchDog:
			// 已经解锁
			return
		}
	}
}

func (l *Lock) Unlock(ctx context.Context) error {
	err := l.script.Run(ctx, l.client, []string{l.resource}, l.randomValue).Err()
	// 关闭看门狗
	close(l.watchDog)
	return err
}
