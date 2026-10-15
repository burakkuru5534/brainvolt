package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
)

// RequestIDHeader carries the request id in both directions (conventions.md §5).
const RequestIDHeader = "X-Request-ID"

// retryAfterTimeout is the Retry-After hint sent when a handler ran out of time.
const retryAfterTimeout = time.Second

// recorder captures the status for logging and remembers whether the
// response has started. Unwrap keeps http.ResponseController working
// (flush, deadlines, hijack for WebSocket upgrades).
type recorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rec *recorder) WriteHeader(code int) {
	if rec.status == 0 {
		rec.status = code
	}
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *recorder) Write(b []byte) (int, error) {
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
	n, err := rec.ResponseWriter.Write(b)
	rec.bytes += n
	return n, err
}

func (rec *recorder) Unwrap() http.ResponseWriter { return rec.ResponseWriter }

func (rec *recorder) started() bool { return rec.status != 0 }

// routeInfo is filled in by the route wrapper so outer middleware can log
// the matched pattern (ServeMux sets Request.Pattern only on its own copy).
type routeInfo struct{ pattern string }

type routeInfoKey struct{}

func routeFrom(ctx context.Context) *routeInfo {
	ri, _ := ctx.Value(routeInfoKey{}).(*routeInfo)
	return ri
}

// securityHeaders sets the headers of conventions.md §13 and makes
// responses non-cacheable unless a handler says otherwise (§11).
func securityHeaders(hsts bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Cache-Control", "no-store")
			if hsts {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requestID accepts a client X-Request-ID only if it is a canonical UUID,
// otherwise generates a UUIDv7; it is echoed and stored in the context.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if _, err := ids.Parse(id); err != nil {
			id = ids.NewString()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(logging.WithRequestID(r.Context(), id)))
	})
}

// locale stores the negotiated locale; without a usable Accept-Language the
// default applies until the identity layer knows the profile locale.
func locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l, ok := i18n.Negotiate(r.Header.Get("Accept-Language"))
		if !ok {
			l = i18n.Default
		}
		next.ServeHTTP(w, r.WithContext(i18n.WithLocale(r.Context(), l)))
	})
}

// accessLog writes one line per request and recovers panics so a crashing
// handler answers 500 INTERNAL instead of dropping the connection.
func accessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &recorder{ResponseWriter: w}
			ri := &routeInfo{}
			r = r.WithContext(context.WithValue(r.Context(), routeInfoKey{}, ri))

			defer func(ctx context.Context) {
				if v := recover(); v != nil {
					if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
						panic(v) // deliberate abort: let net/http drop the connection
					}
					err := fmt.Errorf("panic: %v", v)
					logger.ErrorContext(ctx, "handler panicked", "error", err, "stack", string(debug.Stack()))
					if !rec.started() {
						WriteError(rec, r, logger, apierror.Wrap(apierror.Internal, err))
					}
				}
				level := slog.LevelInfo
				if rec.status >= http.StatusInternalServerError {
					level = slog.LevelError
				}
				logger.LogAttrs(ctx, level, "request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("route", ri.pattern),
					slog.Int("status", rec.status),
					slog.Int("bytes", rec.bytes),
					slog.Duration("duration", time.Since(start)),
				)
			}(r.Context())
			next.ServeHTTP(rec, r)
		})
	}
}

// notFound answers unmatched requests with the error envelope: 404, or 405
// with the Allow header when the path exists for other methods.
func notFound(mux *http.ServeMux, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := mux.Handler(r)
		if pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		// Let ServeMux decide between 404 and 405 without writing its plain-text body.
		probe := &headerProbe{header: http.Header{}}
		h.ServeHTTP(probe, r)
		if probe.status == http.StatusMethodNotAllowed {
			w.Header().Set("Allow", probe.header.Get("Allow"))
			WriteError(w, r, logger, apierror.New(apierror.MethodNotAllowed))
			return
		}
		WriteError(w, r, logger, apierror.New(apierror.NotFound))
	})
}

// headerProbe records the status and headers of ServeMux's built-in
// 404/405 handlers and discards their body.
type headerProbe struct {
	header http.Header
	status int
}

func (p *headerProbe) Header() http.Header         { return p.header }
func (p *headerProbe) Write(b []byte) (int, error) { return len(b), nil }
func (p *headerProbe) WriteHeader(code int)        { p.status = code }

// route wraps one registered handler: records the pattern, applies the
// body limit and the handler deadline, and renders returned errors.
func route(pattern string, h HandlerFunc, bodyLimit int64, timeout time.Duration, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ri := routeFrom(r.Context()); ri != nil {
			ri.pattern = pattern
		}
		if r.ContentLength > bodyLimit {
			WriteError(w, r, logger, apierror.New(apierror.PayloadTooLarge).With("limit_bytes", bodyLimit))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, bodyLimit)

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		r = r.WithContext(ctx)

		if err := h(w, r); err != nil {
			if rec, ok := w.(*recorder); ok && rec.started() {
				logger.ErrorContext(ctx, "handler returned an error after writing the response", "error", err)
				return
			}
			// A handler that hit the deadline may report it as either context error.
			if errors.Is(ctx.Err(), context.DeadlineExceeded) &&
				(errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
				err = fmt.Errorf("handler exceeded %s: %w", timeout, context.DeadlineExceeded)
			}
			WriteError(w, r, logger, err)
		}
	})
}
