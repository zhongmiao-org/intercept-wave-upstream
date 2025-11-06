package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"intercept-wave-upstream/internal/httpserver"
	"intercept-wave-upstream/internal/wsserver"
)

func main() {
	base := httpserver.BasePortFromEnv()
	httpServers := httpserver.StartAll(base)
	wsServers := wsserver.StartAll(base)

	fmt.Printf("Upstream servers started: HTTP:%d-%d WS:%d-%d\n", base, base+2, base+3, base+5)
	// graceful shutdown on SIGINT/SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, s := range httpServers {
		_ = s.Shutdown(ctx)
	}
	for _, s := range wsServers {
		_ = s.Shutdown(ctx)
	}
}
