package pkg

import (
	"context"
	"fmt"
	"goapp/pkg/distribute"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

var locker *distribute.Locker

func init() {
	var err error
	locker, err = distribute.NewLocker(context.Background(), &redis.Options{
		Addr: "127.0.0.1:6379",
		DB:   2,
	})
	if err != nil {
		panic(err)
	}
}

func TestRedLock(t *testing.T) {
	lock, err := locker.Lock(t.Context(), "test", distribute.LockWithTtl(10*time.Second))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer lock.Unlock(t.Context())
	time.Sleep(30 * time.Second)
	fmt.Println("done")
}
