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
			if err := c.WriteMessage(t, msg); err != nil {
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
		key := eventKeyForService(sp)
		delay := parseInterval(r, 300)
		seq := loadFoodFlow("food_user.json", defaultFoodUserFlow(), key)
		for _, m := range seq {
			if err := writeJSON(c, m); err != nil {
				log.Printf("food user write: %v", err)
				break
			}
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		_ = c.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(time.Second))
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
		key := eventKeyForService(sp)
		delay := parseInterval(r, 450)
		seq := loadFoodFlow("food_merchant.json", defaultFoodMerchantFlow(), key)
		for _, m := range seq {
			if err := writeJSON(c, m); err != nil {
				log.Printf("food merchant write: %v", err)
				break
			}
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		_ = c.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(time.Second))
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
func writeJSON(c *websocket.Conn, m map[string]any) error {
	// use common.jsonMarshal for consistency
	b, err := common.JsonMarshalCompat(m)
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, b)
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
