package lock

import (
	"context"
	"errors"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/go-redis/redis/v8"
)

// Locker 可以通过它得到一把锁
// 它主要是用于配置锁
type Locker struct {
	client          *redis.Client // Redis客户端
	script          *redis.Script // 解锁脚本
	ttl             time.Duration // 过期时间
	tryLockInterval time.Duration // 重新获取锁间隔
}

func NewLocker(client *redis.Client, ttl, tryLockInterval time.Duration) *Locker {
	return &Locker{
		client:          client,
		script:          redis.NewScript(unlockScript),
		ttl:             ttl,
		tryLockInterval: tryLockInterval,
	}
}

func (l *Locker) GetLock(resource string) *Lock {
	return &Lock{
		client:          l.client,
		script:          l.script,
		resource:        resource,
		randomValue:     gofakeit.UUID(),
		watchDog:        make(chan struct{}),
		ttl:             l.ttl,
		tryLockInterval: l.tryLockInterval,
	}
}

// Lock 不可重复使用
type Lock struct {
	client          *redis.Client // Redis客户端
	script          *redis.Script // 解锁脚本
	resource        string        // 锁定的资源
	randomValue     string        // 随机值
	watchDog        chan struct{} // 看门狗
	ttl             time.Duration // 过期时间
	tryLockInterval time.Duration // 重新获取锁间隔
}

func (l *Lock) Lock(ctx context.Context) error {
	// 尝试加锁
	err := l.TryLock(ctx)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrLockFailed) {
		return err
	}
	// 加锁失败，不断尝试
	ticker := time.NewTicker(l.tryLockInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// 超时
			return ErrTimeout
		case <-ticker.C:
			// 重新尝试加锁
			err := l.TryLock(ctx)
			if err == nil {
				return nil
			}
			if !errors.Is(err, ErrLockFailed) {
				return err
			}
		}
	}
}

func (l *Lock) TryLock(ctx context.Context) error {
	success, err := l.client.SetNX(ctx, l.resource, l.randomValue, l.ttl).Result()
	if err != nil {
		return err
	}
	// 加锁失败
	if !success {
		return ErrLockFailed
	}
	// 加锁成功，启动看门狗
	go l.startWatchDog()
	return nil
}

func (l *Lock) startWatchDog() {
	ticker := time.NewTicker(l.ttl / 3)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// 延长锁的过期时间
			ctx, _ := context.WithTimeout(context.Background(), l.ttl/3*2)
			ok, err := l.client.Expire(ctx, l.resource, l.ttl).Result()
			// 异常或锁已经不存在则不再续期
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
