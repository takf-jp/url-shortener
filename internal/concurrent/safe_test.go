package concurrent

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGo_panicしてもプロセスが落ちない(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	Go(context.Background(), func() {
		defer wg.Done()
		panic("test panic")
	})

	// goroutineが終了するのを待つ
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 正常終了
	case <-time.After(1 * time.Second):
		t.Fatal("goroutine did not finish in time")
	}
}
