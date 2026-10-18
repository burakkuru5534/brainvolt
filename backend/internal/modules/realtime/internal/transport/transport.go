// Package transport exposes the WebSocket endpoint (websocket.md §2).
package transport

import (
	"net/http"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/wsserver"
)

// Register adds GET /v1/ws. Authentication happens inside the socket
// (sys.auth within 5 s), so the route itself is public.
func Register(s *httpserver.Server, hub *service.Hub) {
	s.Handle("GET /v1/ws", func(w http.ResponseWriter, r *http.Request) error {
		conn, err := wsserver.Accept(w, r)
		if err != nil {
			return nil //nolint:nilerr // Accept has written the HTTP error response
		}
		hub.Serve(r.Context(), conn, i18n.FromContext(r.Context()))
		return nil
	}, httpserver.LongLived(), httpserver.WithDoc(httpserver.Doc{
		Tag: "realtime", Auth: "public", Status: http.StatusSwitchingProtocols,
		Summary: "WebSocket upgrade; the first message must be sys.auth within 5 s (docs/api/websocket.md)",
	}))
}
