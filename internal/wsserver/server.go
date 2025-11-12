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

// staticWsToken is the hardcoded token required to access WS endpoints.
// Clients must provide it via header "X-Auth-Token" or query param "token".
const staticWsToken = "zhongmiao-org-token"

func requireToken(w http.ResponseWriter, r *http.Request) bool {
	tok := r.Header.Get("X-Auth-Token")
	if tok == "" {
		tok = r.URL.Query().Get("token")
	}
	if tok != staticWsToken {
		common.JSON(w, http.StatusUnauthorized, map[string]any{
			"error": "unauthorized",
			"hint":  "provide X-Auth-Token header or ?token=",
		})
		return false
	}
	return true
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
		if !requireToken(w, r) {
			return
		}
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
			logWsFrame(sp, "recv", t, msg)
			if err := writeMessageLogged(c, sp, t, msg); err != nil {
				log.Printf("write echo: %v", err)
				return
			}
		}
	})

	mux.HandleFunc("/ws/ticker", func(w http.ResponseWriter, r *http.Request) {
		if !requireToken(w, r) {
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		defer func() { _ = c.Close() }()
		// start background reader to log any inbound messages; signals done on error/close
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				t, msg, err := c.ReadMessage()
				if err != nil {
					// connection closed or fatal read; stop
					common.Logf("WS %s recv loop end: %v", sp.Name, err)
					return
				}
				logWsFrame(sp, "recv", t, msg)
			}
		}()
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
				if err := writeMessageLogged(c, sp, websocket.TextMessage, []byte(fmt.Sprintf("tick %d", i))); err != nil {
					log.Printf("write ticker: %v", err)
					return
				}
			case <-done:
				// client closed or read failed; stop sending
				return
			}
		}
	})

	mux.HandleFunc("/ws/timeline", func(w http.ResponseWriter, r *http.Request) {
		if !requireToken(w, r) {
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		defer func() { _ = c.Close() }()
		// Load timeline messages from assets if present
		msgs := []string{"hello", "processing", "done"}
		if v, err := common.LoadJSONDynamic(common.JoinAssets("ws", "timeline.json")); err == nil {
			if arr, ok := v.([]any); ok && len(arr) > 0 {
				tmp := make([]string, 0, len(arr))
				for _, it := range arr {
					if s, ok := it.(string); ok {
						tmp = append(tmp, s)
					}
				}
				if len(tmp) > 0 {
					msgs = tmp
				}
			}
		}
		// background reader: log and stop sequence when client closes
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				t, msg, err := c.ReadMessage()
				if err != nil {
					common.Logf("WS %s recv loop end: %v", sp.Name, err)
					return
				}
				logWsFrame(sp, "recv", t, msg)
			}
		}()
	timelineLoop:
		for _, m := range msgs {
			if err := writeMessageLogged(c, sp, websocket.TextMessage, []byte(m)); err != nil {
				log.Printf("write timeline: %v", err)
				break
			}
			time.Sleep(300 * time.Millisecond)
			select {
			case <-done:
				// stop sequence when read loop ends
				break timelineLoop
			default:
			}
		}
		if err := writeControlLogged(c, sp, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(time.Second)); err != nil {
			log.Printf("write close ctrl: %v", err)
		}
	})

	// Food delivery workflow simulation endpoints
	// - /ws/food/user: end-user notifications
	// - /ws/food/merchant: merchant-side notifications
	mux.HandleFunc("/ws/food/user", func(w http.ResponseWriter, r *http.Request) {
		if !requireToken(w, r) {
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		defer func() { _ = c.Close() }()
		// background reader to log inbound
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				t, msg, err := c.ReadMessage()
				if err != nil {
					common.Logf("WS %s recv loop end: %v", sp.Name, err)
					return
				}
				logWsFrame(sp, "recv", t, msg)
			}
		}()
		key := eventKeyForService(sp)
		delay := parseInterval(r, 300)
		seq := loadFoodFlow("food_user.json", defaultFoodUserFlow(), key)
	userLoop:
		for _, m := range seq {
			if err := writeJSONWithLog(c, sp, m); err != nil {
				log.Printf("food user write: %v", err)
				break
			}
			time.Sleep(time.Duration(delay) * time.Millisecond)
			select {
			case <-done:
				break userLoop
			default:
			}
		}
		_ = writeControlLogged(c, sp, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(time.Second))
	})

	mux.HandleFunc("/ws/food/merchant", func(w http.ResponseWriter, r *http.Request) {
		if !requireToken(w, r) {
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		defer func() { _ = c.Close() }()
		// background reader to log inbound
		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				t, msg, err := c.ReadMessage()
				if err != nil {
					common.Logf("WS %s recv loop end: %v", sp.Name, err)
					return
				}
				logWsFrame(sp, "recv", t, msg)
			}
		}()
		key := eventKeyForService(sp)
		delay := parseInterval(r, 450)
		seq := loadFoodFlow("food_merchant.json", defaultFoodMerchantFlow(), key)
	merchantLoop:
		for _, m := range seq {
			if err := writeJSONWithLog(c, sp, m); err != nil {
				log.Printf("food merchant write: %v", err)
				break
			}
			time.Sleep(time.Duration(delay) * time.Millisecond)
			select {
			case <-done:
				break merchantLoop
			default:
			}
		}
		_ = writeControlLogged(c, sp, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(time.Second))
	})
}

// pick event key per WS service: ensure diversity across the three services
func eventKeyForService(sp WsSpec) string {
	switch sp.Name {
	case "ws-echo":
		return "type"
	case "ws-ticker":
		return "action"
	case "ws-timeline":
		return "event"
	default:
		return "type"
	}
}

// parseInterval reads ?interval=ms (default def)
func parseInterval(r *http.Request, def int) int {
	ivalStr := r.URL.Query().Get("interval")
	ival, _ := strconv.Atoi(ivalStr)
	if ival <= 0 {
		ival = def
	}
	return ival
}

// writeJSON serializes a map to JSON text message
func writeJSONWithLog(c *websocket.Conn, sp WsSpec, m map[string]any) error {
	// use common.jsonMarshal for consistency
	b, err := common.JsonMarshalCompat(m)
	if err != nil {
		return err
	}
	return writeMessageLogged(c, sp, websocket.TextMessage, b)
}

// writeMessageLogged writes a WS frame and logs the payload direction/type.
func writeMessageLogged(c *websocket.Conn, sp WsSpec, t int, payload []byte) error {
	logWsFrame(sp, "send", t, payload)
	return c.WriteMessage(t, payload)
}

// writeControlLogged writes a control frame (e.g., close) and logs it.
func writeControlLogged(c *websocket.Conn, sp WsSpec, t int, payload []byte, deadline time.Time) error {
	logWsFrame(sp, "send", t, payload)
	return c.WriteControl(t, payload, deadline)
}

// logWsFrame prints a concise representation of a WS frame.
func logWsFrame(sp WsSpec, dir string, t int, payload []byte) {
	tp := wsTypeName(t)
	msg := summarizePayload(t, payload)
	common.Logf("WS %s %s [%s]: %s", sp.Name, dir, tp, msg)
}

func wsTypeName(t int) string {
	switch t {
	case websocket.TextMessage:
		return "text"
	case websocket.BinaryMessage:
		return "binary"
	case websocket.CloseMessage:
		return "close"
	case websocket.PingMessage:
		return "ping"
	case websocket.PongMessage:
		return "pong"
	default:
		return fmt.Sprintf("type_%d", t)
	}
}

func summarizePayload(t int, b []byte) string {
	if t == websocket.TextMessage || t == websocket.PingMessage || t == websocket.PongMessage || t == websocket.CloseMessage {
		const limit = 200
		s := string(b)
		if len(s) > limit {
			return fmt.Sprintf("%q...(truncated %d bytes)", s[:limit], len(s)-limit)
		}
		return fmt.Sprintf("%q", s)
	}
	// binary: don't print raw bytes to keep logs readable
	return fmt.Sprintf("%d bytes", len(b))
}

// default flows (user/merchant)
func defaultFoodUserFlow() []map[string]any {
	return []map[string]any{
		{"type": "order_accepted", "orderId": "123456", "riderId": "rider_888", "riderName": "张师傅", "riderPhone": "138****1234", "estimatedArrival": "19:30"},
		{"type": "order_picked_up", "orderId": "123456", "timestamp": 1621234567890},
		{"type": "rider_location_update", "orderId": "123456", "location": map[string]any{"lat": 39.9042, "lng": 116.4074}},
		{"type": "rider_location_update", "orderId": "123456", "location": map[string]any{"lat": 39.9050, "lng": 116.4080}},
		{"type": "order_delivered", "orderId": "123456", "timestamp": 1621234667890},
		{"type": "order_cancelled", "orderId": "123456", "reason": "用户取消"},
	}
}

func defaultFoodMerchantFlow() []map[string]any {
	return []map[string]any{
		{"type": "new_order", "orderId": "123456", "items": []any{map[string]any{"name": "红烧牛肉面", "qty": 2}}, "userAddress": "XX路XX号", "userNote": "不要香菜"},
		{"type": "order_cancelled", "orderId": "123456", "reason": "商家拒单"},
	}
}

// loadFoodFlow loads an array of objects from assets/ws/{file}; if missing, uses fallback.
// Then ensures the key name matches the desired event key by renaming when needed.
func loadFoodFlow(file string, fallback []map[string]any, key string) []map[string]any {
	path := common.JoinAssets("ws", file)
	if v, err := common.LoadJSONDynamic(path); err == nil {
		if arr, ok := v.([]any); ok {
			out := make([]map[string]any, 0, len(arr))
			for _, it := range arr {
				if m, ok := it.(map[string]any); ok {
					out = append(out, normalizeEventKey(m, key))
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	// apply key normalization to fallback too
	out := make([]map[string]any, 0, len(fallback))
	for _, m := range fallback {
		out = append(out, normalizeEventKey(m, key))
	}
	return out
}

// normalizeEventKey renames known keys to the desired key when necessary.
func normalizeEventKey(m map[string]any, key string) map[string]any {
	if key == "type" {
		return m
	}
	cp := map[string]any{}
	for k, v := range m {
		cp[k] = v
	}
	// prefer existing explicit key if already matches
	if _, ok := cp[key]; ok {
		return cp
	}
	// map common fields
	if v, ok := cp["type"]; ok {
		cp[key] = v
		delete(cp, "type")
		return cp
	}
	if v, ok := cp["event"]; ok {
		cp[key] = v
		delete(cp, "event")
		return cp
	}
	if v, ok := cp["action"]; ok {
		cp[key] = v
		delete(cp, "action")
		return cp
	}
	// nothing to rename
	return cp
}
