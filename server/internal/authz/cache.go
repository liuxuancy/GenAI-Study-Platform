// 进程内权限缓存：按用户 ID 缓存 Principal，过期或 Reset 后重新加载。

package authz

import (
	"context"
	"sync"
	"time"
)

// PrincipalLoader 按用户 ID 加载权限快照。正式实现查数据库，测试实现返回预设数据。
type PrincipalLoader interface {
	LoadPrincipal(ctx context.Context, userID int64) (*Principal, error)
}

// Cache 进程内的权限缓存，按用户 ID 存放快照，过期或 Reset 后重新加载。
type Cache struct {
	ttl    time.Duration    // 每条缓存的有效期
	loader PrincipalLoader  // 未命中时用来查数据库
	now    func() time.Time // 测试时替换成假时钟

	mutex   sync.Mutex           // 保护下面两个字段，多个请求会同时读写
	entries map[int64]cacheEntry // 用户 ID → 权限快照和过期时间
	version uint64               // 每次 Reset 加一，用来丢弃 Reset 之前发起的加载结果
}

type cacheEntry struct {
	principal *Principal
	expiresAt time.Time
}

func NewCache(ttl time.Duration, loader PrincipalLoader) *Cache {
	return &Cache{
		ttl:     ttl,
		loader:  loader,
		now:     time.Now,
		entries: make(map[int64]cacheEntry),
	}
}

// Get 返回用户的权限快照，未命中或已过期时从数据库加载。加载失败不缓存。
func (cache *Cache) Get(ctx context.Context, userID int64) (*Principal, error) {
	if principal, ok := cache.lookup(userID); ok {
		return principal, nil
	}

	version := cache.getCurrentVersion()
	principal, err := cache.loader.LoadPrincipal(ctx, userID) // 不持锁加载，避免一个慢查询卡住所有请求
	if err != nil {
		return nil, err
	}
	cache.store(userID, principal, version)
	return principal, nil
}

// Reset 清空全部缓存。任何影响权限的写操作提交后都要调用。
func (cache *Cache) Reset() {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.entries = make(map[int64]cacheEntry)
	cache.version++
}

// 查找缓存中userID对应的权限快照
func (cache *Cache) lookup(userID int64) (*Principal, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	entry, exists := cache.entries[userID]
	if !exists {
		return nil, false
	}
	// 到达过去时刻，缓存已经过期
	if !cache.now().Before(entry.expiresAt) {
		delete(cache.entries, userID)
		return nil, false
	}
	return entry.principal, true
}

func (cache *Cache) getCurrentVersion() uint64 {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	return cache.version
}

// store 只在加载期间没有发生 Reset 时写入，否则这份快照可能是旧权限。
func (cache *Cache) store(userID int64, principal *Principal, version uint64) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if version != cache.version {
		return
	}
	cache.entries[userID] = cacheEntry{principal: principal, expiresAt: cache.now().Add(cache.ttl)}
}
