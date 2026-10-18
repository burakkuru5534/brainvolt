// Package wsserver is the WebSocket transport of docs/api/websocket.md
// (ADR-0002: coder/websocket): upgrading a request, the JSON envelopes,
// the per-message size limit, the per-connection message rate and the
// close codes. The protocol itself (auth, heartbeat, routing, resume) lives
// in the realtime module.
package wsserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Limits (websocket.md §1, limits.md, D-032).
const (
	// MaxMessageBytes caps one client message.
	MaxMessageBytes = 4 << 10
	// MessagesPerSecond caps client messages per connection.
	MessagesPerSecond = 20
	// hardReadLimit is where the library drops the connection itself.
	hardReadLimit = 64 << 10
	// WriteTimeout bounds one frame write; a client that cannot take a
	// frame in this time is dropped.
	WriteTimeout = 5 * time.Second
)

// CloseCode is a WebSocket close status.
type CloseCode = websocket.StatusCode

// Close codes (websocket.md §2.1).
const (
	CloseNormal         CloseCode = 1000
	CloseGoingAway      CloseCode = 1001
	CloseProtocol       CloseCode = 4000
	CloseAuthTimeout    CloseCode = 4001
	CloseInvalidToken   CloseCode = 4002
	CloseTokenExpired   CloseCode = 4003
	CloseAccountBlocked CloseCode = 4004
	CloseRateLimited    CloseCode = 4008
	CloseReplaced       CloseCode = 4009
	CloseVersion        CloseCode = 4026
)

// Inbound is a client → server message (websocket.md §3).
type Inbound struct {
	T  string          `json:"t"`
	ID string          `json:"id,omitempty"`
	D  json.RawMessage `json:"d,omitempty"`
}

// Outbound is a server → client message (websocket.md §3).
type Outbound struct {
	T   string `json:"t"`
	S   string `json:"s,omitempty"`
	Seq int64  `json:"seq,omitempty"`
	TS  int64  `json:"ts"`
	Re  string `json:"re,omitempty"`
	D   any    `json:"d,omitempty"`
}

// CloseError is returned by Read when the connection should be closed
// with a protocol code.
type CloseError struct {
	Code   CloseCode
	Reason string
}

func (e *CloseError) Error() string { return fmt.Sprintf("websocket close %d: %s", e.Code, e.Reason) }

// Conn is one accepted connection. Read is called by one goroutine; Write
// and Close are safe for concurrent use.
type Conn struct {
	c *websocket.Conn

	window   time.Time // start of the current one-second rate window
	inWindow int

	closeOnce sync.Once
}

// Accept upgrades the request. Native apps send no Origin header; browser
// origins other than the request host are refused by the library.
func Accept(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return nil, err
	}
	// The library closes with 1009 over its limit; below it, Read answers
	// an oversized message with the protocol's own 4000 (websocket.md §2.1).
	c.SetReadLimit(hardReadLimit)
	return &Conn{c: c}, nil
}

// Read returns the next client message. A message that is too large, not
// JSON text or over the rate limit returns a *CloseError; any other error
// means the connection is gone.
func (c *Conn) Read(ctx context.Context, now time.Time) (Inbound, error) {
	typ, data, err := c.c.Read(ctx)
	if err != nil {
		return Inbound{}, err
	}
	if len(data) > MaxMessageBytes {
		return Inbound{}, &CloseError{Code: CloseProtocol, Reason: "message too large"}
	}
	if now.Sub(c.window) >= time.Second {
		c.window, c.inWindow = now, 0
	}
	c.inWindow++
	if c.inWindow > MessagesPerSecond {
		return Inbound{}, &CloseError{Code: CloseRateLimited, Reason: "rate limit exceeded"}
	}
	var m Inbound
	if typ != websocket.MessageText || json.Unmarshal(data, &m) != nil || m.T == "" {
		return Inbound{}, &CloseError{Code: CloseProtocol, Reason: "malformed message"}
	}
	return m, nil
}

// Write sends one frame.
func (c *Conn) Write(ctx context.Context, m Outbound) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, WriteTimeout)
	defer cancel()
	return c.c.Write(ctx, websocket.MessageText, data)
}

// Close closes the connection with code; later calls do nothing.
func (c *Conn) Close(code CloseCode, reason string) {
	c.closeOnce.Do(func() {
		if err := c.c.Close(code, reason); err != nil {
			_ = c.c.CloseNow() // the close handshake failed: drop the connection
		}
	})
}
