package app

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
)

// wsClient is a test WebSocket client speaking websocket.md.
type wsClient struct {
	t      *testing.T
	c      *websocket.Conn
	frames chan map[string]any
	closed chan websocket.StatusCode
	// pong answers sys.ping automatically.
	pong bool
	n    int
}

func (x *e2e) url() string {
	x.t.Helper()
	if x.httpURL == "" {
		ts := httptest.NewServer(x.srv.Handler())
		x.t.Cleanup(ts.Close)
		ctx, cancel := context.WithCancel(context.Background())
		go func() { _ = x.mods.realtime.Run(ctx) }()
		x.t.Cleanup(cancel)
		x.httpURL = ts.URL
	}
	return x.httpURL
}

func (x *e2e) dial() *wsClient {
	x.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(x.url(), "http")+"/v1/ws", nil)
	if err != nil {
		x.t.Fatal(err)
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}
	w := &wsClient{t: x.t, c: c, frames: make(chan map[string]any, 64), closed: make(chan websocket.StatusCode, 1), pong: true}
	go w.readLoop()
	x.t.Cleanup(func() { _ = c.CloseNow() })
	return w
}

func (w *wsClient) readLoop() {
	for {
		_, data, err := w.c.Read(context.Background())
		if err != nil {
			w.closed <- websocket.CloseStatus(err)
			close(w.frames)
			return
		}
		var f map[string]any
		_ = json.Unmarshal(data, &f)
		if f["t"] == "sys.ping" && w.pong {
			d, _ := f["d"].(map[string]any)
			w.send("sys.pong", map[string]any{"ts": d["ts"], "client_ts": time.Now().UnixMilli()})
			continue
		}
		w.frames <- f
	}
}

func (w *wsClient) send(t string, d any) string {
	w.n++
	id := "c-" + string(rune('a'+w.n%26)) + ids.NewString()[:8]
	b, _ := json.Marshal(map[string]any{"t": t, "id": id, "d": d})
	_ = w.c.Write(context.Background(), websocket.MessageText, b)
	return id
}

// next returns the next frame of type typ (others are skipped).
func (w *wsClient) next(typ string) map[string]any {
	w.t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case f, ok := <-w.frames:
			if !ok {
				w.t.Fatalf("connection closed while waiting for %s", typ)
			}
			if f["t"] == typ {
				return f
			}
		case <-timeout:
			w.t.Fatalf("no %s within 5 s", typ)
		}
	}
}

func (w *wsClient) waitClose() websocket.StatusCode {
	w.t.Helper()
	select {
	case code := <-w.closed:
		return code
	case <-time.After(7 * time.Second):
		w.t.Fatal("connection not closed")
		return 0
	}
}

func data(f map[string]any) map[string]any { d, _ := f["d"].(map[string]any); return d }

// signedIn returns a token of a registered guest.
func (x *e2e) signedIn() (token string, user uuid.UUID) {
	x.t.Helper()
	token, _ = x.emu.signUpAnonymous()
	r := x.session(token, nil)
	if r.code != 200 {
		x.t.Fatalf("session: %d %s", r.code, r.raw)
	}
	id, _ := r.get("user", "id").(string)
	return token, uuid.MustParse(id)
}

func (x *e2e) auth(token, installation string, resume map[string]any) (*wsClient, map[string]any) {
	x.t.Helper()
	w := x.dial()
	d := map[string]any{"token": token, "installation_id": installation}
	if resume != nil {
		d["resume"] = resume
	}
	w.send("sys.auth", d)
	return w, data(w.next("sys.welcome"))
}

func TestWebSocketHandshake(t *testing.T) {
	x := newE2E(t)
	token, user := x.signedIn()

	// Not sys.auth first → 4001; bad token → 4002; garbage → 4000.
	w := x.dial()
	w.send("match.ready", map[string]any{})
	if code := w.waitClose(); code != 4001 {
		t.Fatalf("wrong first message: close %d", code)
	}
	w = x.dial()
	w.send("sys.auth", map[string]any{"token": "nope", "installation_id": "i1"})
	if code := w.waitClose(); code != 4002 {
		t.Fatalf("bad token: close %d", code)
	}
	w = x.dial()
	_ = w.c.Write(context.Background(), websocket.MessageText, []byte("{not json"))
	if code := w.waitClose(); code != 4000 {
		t.Fatalf("garbage: close %d", code)
	}

	w, welcome := x.auth(token, "inst-a", nil)
	if welcome["user_id"] != user.String() || welcome["session_id"] == "" || welcome["resume_token"] == "" ||
		welcome["heartbeat_ms"] != float64(20_000) {
		t.Fatalf("welcome %v", welcome)
	}
	// Unknown types are ignored; reauth is acknowledged.
	w.send("nothing.here", nil)
	id := w.send("sys.reauth", map[string]any{"token": token})
	if f := w.next("sys.ack"); f["re"] != id {
		t.Fatalf("reauth ack %v", f)
	}

	// Oversized message → 4000.
	w.send("chat.send", map[string]any{"body": strings.Repeat("x", 5000)})
	if code := w.waitClose(); code != 4000 {
		t.Fatalf("oversized: close %d", code)
	}

	// 20 messages per second at most → 4008.
	w, _ = x.auth(token, "inst-a", nil)
	for range 25 {
		w.send("presence.set", map[string]any{"status": "online"})
	}
	if code := w.waitClose(); code != 4008 {
		t.Fatalf("flood: close %d", code)
	}
}

func TestWebSocketRoutingDeliveryAndResume(t *testing.T) {
	x := newE2E(t)
	var mu sync.Mutex
	var resumed []string
	var pongs int
	x.mods.realtime.Handle("test", func(_ context.Context, c realtime.Client, m realtime.Message) error {
		if m.Type == "test.fail" {
			return apierror.New(apierror.MatchStateInvalid)
		}
		return nil
	})
	x.mods.realtime.OnResume("test", func(ctx context.Context, c realtime.Client, stream string, last int64) error {
		mu.Lock()
		resumed = append(resumed, stream)
		mu.Unlock()
		return x.mods.realtime.Send(ctx, realtime.Target{User: c.UserID, Session: c.SessionID},
			realtime.Frame{T: "test.replayed", S: stream, Seq: last + 1})
	})
	x.mods.realtime.OnPong(func(realtime.Client, int) { mu.Lock(); pongs++; mu.Unlock() })
	token, user := x.signedIn()

	w, welcome := x.auth(token, "inst-a", nil)
	id := w.send("test.ok", nil)
	if f := w.next("sys.ack"); f["re"] != id {
		t.Fatalf("ack %v", f)
	}
	id = w.send("test.fail", nil)
	f := w.next("sys.error")
	if f["re"] != id || data(f)["code"] != string(apierror.MatchStateInvalid) || data(f)["message"] == "" {
		t.Fatalf("error %v", f)
	}

	// Delivery to the user and to one session.
	other, _ := x.auth(token, "inst-b", nil)
	if err := x.mods.realtime.Send(t.Context(), realtime.Target{User: user}, realtime.Frame{T: "test.all"}); err != nil {
		t.Fatal(err)
	}
	w.next("test.all")
	other.next("test.all")
	sid := uuid.MustParse(welcome["session_id"].(string))
	if err := x.mods.realtime.Send(t.Context(), realtime.Target{User: user, Session: sid}, realtime.Frame{T: "test.one"}); err != nil {
		t.Fatal(err)
	}
	w.next("test.one")

	// In play the heartbeat runs every 2 s; pongs reach the listeners.
	x.mods.realtime.SetPlaying(sid, true)
	time.Sleep(2500 * time.Millisecond)
	mu.Lock()
	gotPong := pongs > 0
	mu.Unlock()
	if !gotPong {
		t.Fatal("no pong sample within 2.5 s in play")
	}

	// A new socket of the same installation replaces the old one (4009);
	// it resumes the session and its streams.
	again, welcome2 := x.auth(token, "inst-a", map[string]any{
		"session_id": welcome["session_id"], "resume_token": welcome["resume_token"], "streams": map[string]any{"test:1": 7},
	})
	if code := w.waitClose(); code != 4009 {
		t.Fatalf("replaced: close %d", code)
	}
	if welcome2["session_id"] != welcome["session_id"] || welcome2["resume_token"] == welcome["resume_token"] {
		t.Fatalf("resume welcome %v", welcome2)
	}
	if r := again.next("test.replayed"); r["s"] != "test:1" || r["seq"] != float64(8) {
		t.Fatalf("replay %v", r)
	}
	// sys.subscribe asks for a stream from a sequence number.
	id = again.send("sys.subscribe", map[string]any{"stream": "test:2", "from_seq": 3})
	if r := again.next("test.replayed"); r["s"] != "test:2" || r["seq"] != float64(3) {
		t.Fatalf("subscribe replay %v", r)
	}
	if f := again.next("sys.ack"); f["re"] != id {
		t.Fatalf("subscribe ack %v", f)
	}
	// A wrong resume token starts a fresh session.
	_, welcome3 := x.auth(token, "inst-c", map[string]any{"session_id": welcome["session_id"], "resume_token": "x"})
	if welcome3["session_id"] == welcome["session_id"] {
		t.Fatal("resumed with a wrong token")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(resumed) != 2 {
		t.Fatalf("resumed %v", resumed)
	}
}
