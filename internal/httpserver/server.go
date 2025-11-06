package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"intercept-wave-upstream/internal/common"
)

type ServiceSpec struct {
	Name            string
	Port            int
	InterceptPrefix string
	Routes          func(mux *http.ServeMux, spec ServiceSpec)
}

func StartAll(base int) []*http.Server {
	services := []ServiceSpec{
		{Name: "user-service", Port: base + 0, InterceptPrefix: "/api", Routes: userRoutes},
		{Name: "order-service", Port: base + 1, InterceptPrefix: "/order-api", Routes: orderRoutes},
		{Name: "payment-service", Port: base + 2, InterceptPrefix: "/pay-api", Routes: paymentRoutes},
	}

	var wg sync.WaitGroup
	servers := make([]*http.Server, 0, len(services))
	for _, s := range services {
		mux := http.NewServeMux()
		attachCommon(mux, s)
		s.Routes(mux, s)
		server := &http.Server{Addr: fmt.Sprintf(":%d", s.Port), Handler: common.RequestLogger(mux)}
		servers = append(servers, server)
		wg.Add(1)
		go func(sp ServiceSpec, srv *http.Server) {
			defer wg.Done()
			common.Logf("HTTP %s listening on :%d", sp.Name, sp.Port)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				common.Logf("HTTP server %s error: %v", sp.Name, err)
			}
		}(s, server)
	}
	// give servers time to bind
	time.Sleep(100 * time.Millisecond)
	return servers
}

func attachCommon(mux *http.ServeMux, spec ServiceSpec) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		common.JSON(w, 200, map[string]any{
			"service":         spec.Name,
			"port":            spec.Port,
			"interceptPrefix": spec.InterceptPrefix,
			"message":         "Upstream running",
		})
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		common.JSON(w, 200, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		codeStr := strings.TrimPrefix(r.URL.Path, "/status/")
		code, _ := strconv.Atoi(codeStr)
		if code < 100 || code > 599 {
			code = 400
		}
		common.JSON(w, code, map[string]any{"status": code})
	})

	mux.HandleFunc("/delay/", func(w http.ResponseWriter, r *http.Request) {
		msStr := strings.TrimPrefix(r.URL.Path, "/delay/")
		ms, _ := strconv.Atoi(msStr)
		if ms < 0 {
			ms = 0
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		common.JSON(w, 200, map[string]any{"delayedMs": ms})
	})

	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		keys := []string{"Authorization", "Content-Type", "User-Agent", "X-Request-Id"}
		m := map[string]string{}
		for _, k := range keys {
			if v := r.Header.Get(k); v != "" {
				m[k] = v
			}
		}
		common.JSON(w, 200, map[string]any{"headers": m})
	})

	mux.HandleFunc("/cookies", func(w http.ResponseWriter, r *http.Request) {
		cs := map[string]string{}
		for _, c := range r.Cookies() {
			cs[c.Name] = c.Value
		}
		common.JSON(w, 200, map[string]any{"cookies": cs})
	})

	mux.HandleFunc("/large", func(w http.ResponseWriter, r *http.Request) {
		szStr := r.URL.Query().Get("size")
		sz, _ := strconv.Atoi(szStr)
		if sz <= 0 {
			sz = 64 * 1024
		}
		if sz > 2*1024*1024 {
			sz = 2 * 1024 * 1024
		}
		buf := make([]byte, sz)
		for i := range buf {
			buf[i] = byte('a' + (i % 26))
		}
		common.JSON(w, 200, map[string]any{"size": sz, "data": string(buf)})
	})

	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		common.JSON(w, 200, map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
			"query":  r.URL.RawQuery,
			"length": len(b),
			"body":   string(b),
		})
	})
}

func userRoutes(mux *http.ServeMux, spec ServiceSpec) {
	p := spec.InterceptPrefix
	mux.HandleFunc(p+"/user/info", func(w http.ResponseWriter, r *http.Request) {
		common.JSON(w, 200, map[string]any{
			"code": 0,
			"data": map[string]any{
				"id":    1,
				"name":  "张三",
				"email": "zhangsan@example.com",
			},
			"message": "success",
		})
	})
	mux.HandleFunc(p+"/posts", func(w http.ResponseWriter, r *http.Request) {
		posts := make([]map[string]any, 0, 5)
		for i := 1; i <= 5; i++ {
			posts = append(posts, map[string]any{
				"id":        i,
				"title":     fmt.Sprintf("Post %d", i),
				"createdAt": time.Now().Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
			})
		}
		common.JSON(w, 200, map[string]any{"code": 0, "data": posts})
	})
}

func orderRoutes(mux *http.ServeMux, spec ServiceSpec) {
	p := spec.InterceptPrefix
	mux.HandleFunc(p+"/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			b, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			var in map[string]any
			_ = json.Unmarshal(b, &in)
			in["id"] = rand.IntN(100000)
			common.JSON(w, 201, map[string]any{"code": 0, "data": in})
			return
		}
		orders := []map[string]any{
			{"id": 1001, "status": "CREATED"},
			{"id": 1002, "status": "PAID"},
		}
		common.JSON(w, 200, map[string]any{"code": 0, "data": orders})
	})
	// emulate wildcard: /order/{id}/submit
	mux.HandleFunc(p+"/order/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/submit") {
			common.JSON(w, 200, map[string]any{"message": "submit ok"})
			return
		}
		http.NotFound(w, r)
	})
}

func paymentRoutes(mux *http.ServeMux, spec ServiceSpec) {
	p := spec.InterceptPrefix
	mux.HandleFunc(p+"/checkout", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		common.JSON(w, 200, map[string]any{
			"code": 0,
			"data": map[string]any{
				"paid":     true,
				"amount":   199,
				"currency": "CNY",
			},
			"message": "paid",
		})
	})
}

func BasePortFromEnv() int {
	if v := os.Getenv("BASE_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 9000
}
