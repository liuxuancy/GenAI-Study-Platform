package access

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testTTL = 30 * time.Second

// fakeClock 可手动拨动的时钟。
type fakeClock struct {
	current time.Time
}

func (clock *fakeClock) Now() time.Time { return clock.current }

func (clock *fakeClock) Advance(duration time.Duration) { clock.current = clock.current.Add(duration) }

// 编译期检查：两个测试实现都满足 PrincipalLoader 接口
var (
	_ PrincipalLoader = (*countingLoader)(nil)
	_ PrincipalLoader = (*resettingLoader)(nil)
)

// countingLoader 记录加载次数，每次返回当前设定的权限。
type countingLoader struct {
	calls       atomic.Int32 // LoadPrincipal 被调用的次数；并发测试会同时加 1，所以用原子类型
	permissions []string     // 下次 LoadPrincipal 返回的权限；测试中途修改它来模拟"数据库里撤销了权限"
	err         error
}

func newCountingLoader(permissions ...string) *countingLoader {
	return &countingLoader{permissions: permissions}
}

func (loader *countingLoader) callCount() int32 { return loader.calls.Load() }

func (loader *countingLoader) LoadPrincipal(ctx context.Context, userID int64) (*Principal, error) {
	loader.calls.Add(1)
	if loader.err != nil {
		return nil, loader.err
	}
	return &Principal{UserID: userID, Role: RoleStudent, Permissions: loader.permissions}, nil
}

func newTestCache(loader PrincipalLoader) (*Cache, *fakeClock) {
	clock := &fakeClock{current: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	cache := NewCache(testTTL, loader)
	cache.now = clock.Now
	return cache, clock
}

func mustGet(t *testing.T, cache *Cache, userID int64) *Principal {
	t.Helper()
	principal, err := cache.Get(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func TestCacheHit(t *testing.T) {
	loader := newCountingLoader("student:my:view")
	cache, _ := newTestCache(loader)

	mustGet(t, cache, 1)
	mustGet(t, cache, 1)
	if calls := loader.callCount(); calls != 1 {
		t.Errorf("loader called %d times, want 1", calls)
	}
	mustGet(t, cache, 2) // 不同用户分别缓存
	if calls := loader.callCount(); calls != 2 {
		t.Errorf("loader called %d times, want 2", calls)
	}
}

// 不调用 Reset 时，撤销的权限最迟在 TTL 到期后生效。
func TestCacheExpiresAfterTTL(t *testing.T) {
	loader := newCountingLoader("student:my:view")
	cache, clock := newTestCache(loader)

	if !mustGet(t, cache, 1).HasPermission("student:my:view") {
		t.Fatal("expected permission before revoke")
	}
	loader.permissions = []string{} // 模拟数据库里撤销了权限

	clock.Advance(testTTL - time.Second)
	if !mustGet(t, cache, 1).HasPermission("student:my:view") {
		t.Error("cache should still serve old snapshot before TTL")
	}
	clock.Advance(time.Second)
	if mustGet(t, cache, 1).HasPermission("student:my:view") {
		t.Error("revoked permission still granted after TTL")
	}
}

// 调用 Reset 后，撤销的权限在下一个请求立即生效。
func TestCacheResetTakesEffectImmediately(t *testing.T) {
	loader := newCountingLoader("student:my:view")
	cache, _ := newTestCache(loader)

	mustGet(t, cache, 1)
	loader.permissions = []string{}
	cache.Reset()
	if mustGet(t, cache, 1).HasPermission("student:my:view") {
		t.Error("revoked permission still granted after Reset")
	}
}

func TestCacheDoesNotStoreErrors(t *testing.T) {
	loader := newCountingLoader()
	loader.err = errors.New("db down")
	cache, _ := newTestCache(loader)

	for range 2 {
		if _, err := cache.Get(context.Background(), 1); err == nil {
			t.Fatal("expected error")
		}
	}
	if calls := loader.callCount(); calls != 2 {
		t.Errorf("failed load was cached: loader called %d times, want 2", calls)
	}
}

// resettingLoader 在加载过程中触发一次 Reset，模拟"查库期间管理员改了权限"。
type resettingLoader struct {
	cache *Cache
	calls atomic.Int32 // LoadPrincipal 被调用的次数
}

func (loader *resettingLoader) LoadPrincipal(ctx context.Context, userID int64) (*Principal, error) {
	if loader.calls.Add(1) == 1 {
		loader.cache.Reset()
	}
	return &Principal{UserID: userID}, nil
}

func (loader *resettingLoader) callCount() int32 { return loader.calls.Load() }

func TestCacheDiscardsLoadThatRacedWithReset(t *testing.T) {
	loader := &resettingLoader{}
	cache, _ := newTestCache(loader)
	loader.cache = cache

	mustGet(t, cache, 1) // 加载期间发生了 Reset，结果不能写入缓存
	mustGet(t, cache, 1) // 因此这次必须重新加载
	if calls := loader.callCount(); calls != 2 {
		t.Errorf("stale snapshot was cached: loader called %d times, want 2", calls)
	}
	mustGet(t, cache, 1) // 第二次加载没有被 Reset 打断，应已缓存
	if calls := loader.callCount(); calls != 2 {
		t.Errorf("loader called %d times, want 2", calls)
	}
}

func getRepeatedly(cache *Cache, userID int64, waitGroup *sync.WaitGroup, failures *atomic.Int32) {
	defer waitGroup.Done()
	for range 100 {
		if _, err := cache.Get(context.Background(), userID); err != nil {
			failures.Add(1)
		}
		if userID == 0 {
			cache.Reset()
		}
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	loader := newCountingLoader("student:my:view")
	cache, _ := newTestCache(loader)
	cache.now = time.Now // 假时钟不是并发安全的，这里用真实时钟

	var waitGroup sync.WaitGroup
	var failures atomic.Int32
	for userID := range int64(8) {
		waitGroup.Add(1)
		go getRepeatedly(cache, userID, &waitGroup, &failures)
	}
	waitGroup.Wait()
	if failures.Load() != 0 {
		t.Errorf("%d concurrent Get calls failed", failures.Load())
	}
}
