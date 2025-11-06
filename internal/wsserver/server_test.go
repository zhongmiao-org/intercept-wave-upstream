package wsserver

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func findFreeBase() (int, error) {
	for base := 20010; base < 20100; base += 7 {
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
	return 0, fmt.Errorf("no free port range")
}

func TestWsEcho(t *testing.T) {
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

	// wait a bit for bind
	time.Sleep(150 * time.Millisecond)

	u := fmt.Sprintf("ws://127.0.0.1:%d/ws/echo", base+3)
	c, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = c.Close() }()

	msg := "hello"
	if err := c.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, got, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != msg {
		t.Fatalf("echo mismatch: %q != %q", got, msg)
	}
}
