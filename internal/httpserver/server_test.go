package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

// find a contiguous base port for 6 ports (HTTP: +0..+2, WS would be +3..+5)
func findFreeBase() (int, error) {
	// try several candidates in high range
	for base := 19000; base < 20000; base += 7 {
		ok := true
		var conns []net.Listener
		for p := base; p <= base+5; p++ {
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
			if err != nil {
				ok = false
				break
			}
			conns = append(conns, ln)
		}
		for _, c := range conns {
			_ = c.Close()
		}
		if ok {
			return base, nil
		}
	}
	return 0, fmt.Errorf("no free base port range found")
}

func waitHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func TestHTTPServersStartAndRespond(t *testing.T) {
	base, err := findFreeBase()
	if err != nil {
		t.Fatalf("findFreeBase: %v", err)
	}

	srvs := StartAll(base)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for _, s := range srvs {
			_ = s.Shutdown(ctx)
		}
	})

	// wait for order service
	if err := waitHTTP(fmt.Sprintf("http://127.0.0.1:%d/health", base+1), 2*time.Second); err != nil {
		t.Fatalf("order health: %v", err)
	}

	// call mock path on order service
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/order-api/orders", base+1))
	if err != nil {
		t.Fatalf("GET orders: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["code"] != float64(0) { // json numbers => float64
		t.Fatalf("unexpected code: %v", body["code"])
	}
}
