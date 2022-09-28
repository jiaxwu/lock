package lock

import (
	"context"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

type RedLock struct {
	clients        []*redis.Client // Redis客户端
	successClients []*redis.Client // 加锁成功的客户端
	script         *redis.Script   // 解锁脚本
	resource       string          // 锁定的资源
	randomValue    string          // 随机值
	watchDog       chan struct{}   // 看门狗
}

func NewRedLock(clients []*redis.Client, resource string) *RedLock {
	return &RedLock{
		clients:  clients,
		script:   redis.NewScript(unlockScript),
		resource: resource,
	}
}

func (l *RedLock) TryLock(ctx context.Context) error {
	randomValue := gofakeit.UUID()
	var wg sync.WaitGroup
	wg.Add(len(l.clients))
	// 成功获得锁的Redis实例的客户端
	successClients := make(chan *redis.Client, len(l.clients))
	for _, client := range l.clients {
		go func(client *redis.Client) {
			defer wg.Done()
			success, err := client.SetNX(ctx, l.resource, randomValue, ttl).Result()
			if err != nil {
				return
			}
			// 加锁失败
			if !success {
				return
			}
			// 加锁成功，启动看门狗
			go l.startWatchDog()
			successClients <- client
		}(client)
	}
	// 等待所有获取锁操作完成
	wg.Wait()
	close(successClients)
	// 如果成功加锁得客户端少于客户端数量的一半+1，表示加锁失败
	if len(successClients) < len(l.clients)/2+1 {
		// 就算加锁失败，也要把已经获得的锁给释放掉
		for client := range successClients {
			go func(client *redis.Client) {
				ctx, _ := context.WithTimeout(context.Background(), ttl)
				l.script.Run(ctx, client, []string{l.resource}, randomValue)
			}(client)
		}
		return ErrLockFailed
	}

	// 加锁成功，启动看门狗
	l.randomValue = randomValue
	l.successClients = nil
	for successClient := range successClients {
		l.successClients = append(l.successClients, successClient)
	}

	return nil
}

func (l *RedLock) startWatchDog() {
	l.watchDog = make(chan struct{})
	ticker := time.NewTicker(resetTTLInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// 延长锁的过期时间
			for _, client := range l.successClients {
				go func(client *redis.Client) {
					ctx, _ := context.WithTimeout(context.Background(), ttl-resetTTLInterval)
					client.Expire(ctx, l.resource, ttl)
				}(client)
			}
		case <-l.watchDog:
			// 已经解锁
			return
		}
	}
}

func (l *RedLock) Unlock(ctx context.Context) error {
	for _, client := range l.successClients {
		go func(client *redis.Client) {
			l.script.Run(ctx, client, []string{l.resource}, l.randomValue)
		}(client)
	}
	// 关闭看门狗
	close(l.watchDog)
	return nil
}
