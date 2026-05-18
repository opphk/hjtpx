package redis

import (
	"context"
	"sync"
)

var (
	globalWarmer     *CacheWarmer
	globalWarmerOnce sync.Once
)

func GetGlobalWarmer() *CacheWarmer {
	globalWarmerOnce.Do(func() {
		client := GetClient()
		if client != nil {
			globalWarmer = NewCacheWarmer(client)
			globalWarmer.RegisterDefaultTasks()
		}
	})
	return globalWarmer
}

func InitGlobalWarmer(ctx context.Context) {
	warmer := GetGlobalWarmer()
	if warmer != nil {
		warmer.Start(ctx)
	}
}

func StopGlobalWarmer() {
	if globalWarmer != nil {
		globalWarmer.Stop()
	}
}
