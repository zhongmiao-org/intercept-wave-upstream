package wsserver

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"intercept-wave-upstream/internal/common"

	"github.com/gorilla/websocket"
)

type WsSpec struct {
	Name string
	Port int
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func StartAll(base int) []*http.Server {
	specs := []WsSpec{
		{Name: "ws-echo", Port: base + 3},
		{Name: "ws-ticker", Port: base + 4},
		{Name: "ws-timeline", Port: base + 5},
	}
	servers := make([]*http.Server, 0, len(specs))
	for _, sp := range specs {
		mux := http.NewServeMux()
		attachRoutes(mux, sp)
		srv := &http.Server{Addr: fmt.Sprintf(":%d", sp.Port), Handler: common.RequestLogger(mux)}
		servers = append(servers, srv)
		go func(spec WsSpec, s *http.Server) {
			common.Logf("WS %s listening on :%d", spec.Name, spec.Port)
			if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				common.Logf("WS server %s error: %v", spec.Name, err)
			}
		}(sp, srv)
	}
	// brief wait for bind
	time.Sleep(100 * time.Millisecond)
	return servers
}

func attachRoutes(mux *http.ServeMux, sp WsSpec) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		common.JSON(w, 200, map[string]any{"service": sp.Name, "port": sp.Port})
	})

	mux.HandleFunc("/ws/echo", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		defer func() { _ = c.Close() }()
		for {
			t, msg, err := c.ReadMessage()
			if err != nil {
				break
			}
			if err := c.WriteMessage(t, msg); err != nil {
				log.Printf("write echo: %v", err)
				return
			}
		}
	})

	mux.HandleFunc("/ws/ticker", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		defer func() { _ = c.Close() }()
		ivalStr := r.URL.Query().Get("interval")
		ival, _ := strconv.Atoi(ivalStr)
		if ival <= 0 {
			ival = 1000
		}
		ticker := time.NewTicker(time.Duration(ival) * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-ticker.C:
				i++
				if err := c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("tick %d", i))); err != nil {
					log.Printf("write ticker: %v", err)
					return
				}
			default:
				if err := c.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
					log.Printf("set read deadline: %v", err)
				}
				if _, _, err := c.ReadMessage(); err != nil {
					// continue until client closes; ignore read timeouts
				}
			}
		}
	})

	mux.HandleFunc("/ws/timeline", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		defer func() { _ = c.Close() }()
		msgs := []string{"hello", "processing", "done"}
		for _, m := range msgs {
			if err := c.WriteMessage(websocket.TextMessage, []byte(m)); err != nil {
				log.Printf("write timeline: %v", err)
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
		if err := c.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(time.Second)); err != nil {
			log.Printf("write close ctrl: %v", err)
		}
	})
}
