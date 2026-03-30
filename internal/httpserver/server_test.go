package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// find a contiguous base port for 6 ports (HTTP: +0..+2, WS would be +3..+5)
func findFreeBase() (int, error) {
	// scan a wide high-port range for 6 contiguous ports
	for base := 20000; base < 60000; base += 11 {
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

func decodeJSONBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body
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
	body := decodeJSONBody(t, resp)
	if body["code"] != float64(0) { // json numbers => float64
		t.Fatalf("unexpected code: %v", body["code"])
	}
}

func TestHTTPServersExposeRouteFriendlyAliases(t *testing.T) {
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

	if err := waitHTTP(fmt.Sprintf("http://127.0.0.1:%d/health", base), 2*time.Second); err != nil {
		t.Fatalf("user health: %v", err)
	}

	t.Run("user aliases support strip-prefix forwarding", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/user/info", base))
		if err != nil {
			t.Fatalf("GET /user/info: %v", err)
		}
		body := decodeJSONBody(t, resp)
		data := body["data"].(map[string]interface{})
		if data["email"] != "zhangsan@example.com" {
			t.Fatalf("unexpected email: %v", data["email"])
		}

		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/admin/stats", base))
		if err != nil {
			t.Fatalf("GET /admin/stats: %v", err)
		}
		body = decodeJSONBody(t, resp)
		stats := body["data"].(map[string]interface{})
		if stats["activeUsers"] != float64(128) {
			t.Fatalf("unexpected activeUsers: %v", stats["activeUsers"])
		}

		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/users/42/preferences", base))
		if err != nil {
			t.Fatalf("GET /users/42/preferences: %v", err)
		}
		body = decodeJSONBody(t, resp)
		prefs := body["data"].(map[string]interface{})
		if prefs["userId"] != "42" {
			t.Fatalf("unexpected userId: %v", prefs["userId"])
		}
	})

	t.Run("order aliases include dynamic detail and submit endpoints", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/orders/3009", base+1))
		if err != nil {
			t.Fatalf("GET /orders/3009: %v", err)
		}
		body := decodeJSONBody(t, resp)
		data := body["data"].(map[string]interface{})
		if data["id"] != "3009" {
			t.Fatalf("unexpected order id: %v", data["id"])
		}

		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/admin/orders/summary", base+1))
		if err != nil {
			t.Fatalf("GET /admin/orders/summary: %v", err)
		}
		body = decodeJSONBody(t, resp)
		summary := body["data"].(map[string]interface{})
		if summary["paid"] != float64(9) {
			t.Fatalf("unexpected paid count: %v", summary["paid"])
		}

		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/order/3009/submit", base+1))
		if err != nil {
			t.Fatalf("GET /order/3009/submit: %v", err)
		}
		body = decodeJSONBody(t, resp)
		if body["message"] != "submit ok" {
			t.Fatalf("unexpected message: %v", body["message"])
		}
	})

	t.Run("payment aliases cover preview refunds and callbacks", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/checkout/preview", base+2))
		if err != nil {
			t.Fatalf("GET /checkout/preview: %v", err)
		}
		body := decodeJSONBody(t, resp)
		if body["message"] != "preview" {
			t.Fatalf("unexpected preview message: %v", body["message"])
		}

		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/refunds", base+2))
		if err != nil {
			t.Fatalf("GET /refunds: %v", err)
		}
		body = decodeJSONBody(t, resp)
		refunds := body["data"].([]interface{})
		if len(refunds) < 2 {
			t.Fatalf("unexpected refunds length: %d", len(refunds))
		}

		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/callbacks/alipay", base+2), bytes.NewBufferString(`{"trade_status":"TRADE_SUCCESS"}`))
		if err != nil {
			t.Fatalf("POST /callbacks/alipay request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /callbacks/alipay: %v", err)
		}
		body = decodeJSONBody(t, resp)
		data := body["data"].(map[string]interface{})
		if data["callbackBody"] != `{"trade_status":"TRADE_SUCCESS"}` {
			t.Fatalf("unexpected callback body: %v", data["callbackBody"])
		}
	})
}

func TestPaymentRefundCreateReturnsCreated(t *testing.T) {
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

	if err := waitHTTP(fmt.Sprintf("http://127.0.0.1:%d/health", base+2), 2*time.Second); err != nil {
		t.Fatalf("payment health: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/refunds", base+2), bytes.NewBufferString(`{"orderId":"2002","amount":19.9}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST refunds: %v", err)
	}
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected status=%d", resp.StatusCode)
	}
}
