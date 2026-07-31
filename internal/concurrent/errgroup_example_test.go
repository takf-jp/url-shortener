package concurrent_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

func TestErrgroup_3つのURLを並列GET(t *testing.T) {
	// テスト用のサーバを３つ建てる
	newSlowServer := func(body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond) // 遅延を入れる
			w.Write([]byte(body))
		}))
	}

	srv1, srv2, srv3 := newSlowServer("response from server 1"), newSlowServer("response from server 2"), newSlowServer("response from server 3")

	defer srv1.Close()
	defer srv2.Close()
	defer srv3.Close()

	urls := []string{srv1.URL, srv2.URL, srv3.URL}
	results := make([]string, len(urls)	)

	start := time.Now()
	// errgroupを使って並列にGETリクエストを送る
	g, ctx := errgroup.WithContext(context.Background())
	for i, url := range urls {
		i, url := i, url // ループ変数のキャプチャを避ける
		g.Go(func() error {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			buf := make([]byte, 32)
			n, _ := resp.Body.Read(buf)
			results[i] = string(buf[:n])
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatalf("errgroup failed: %v", err)
	}
	elapsed := time.Since(start)
	t.Logf("elapsed time: %v", elapsed)

	if elapsed >= 200*time.Millisecond {
		t.Errorf("expected pararell execution (<200ms), but took %v", elapsed)
	}

	for i, r := range results {
		if r == "" {
			t.Errorf("expected non-empty response for server %d", i)
		}
	}
}
