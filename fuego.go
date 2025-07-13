package otelfuego

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName    = "github.com/pdrvsky/otelfuego/otelfuego"
	instrumentationVersion = "0.1.0"
)

// Middleware returns a fuego middleware that instruments HTTP requests with OpenTelemetry.
// It provides automatic span creation, request/response instrumentation, and distributed tracing support.
//
// Basic usage:
//
//	server := fuego.NewServer()
//	server.Use(otelfuego.Middleware("my-service"))
//
// With options:
//
//	server.Use(otelfuego.Middleware("my-service",
//	    otelfuego.WithFilter(func(req *http.Request) bool {
//	        return !strings.Contains(req.URL.Path, "/health")
//	    }),
//	))
func Middleware(service string, opts ...Option) func(http.Handler) http.Handler {
	cfg := newConfig(opts...)

	// Get tracer from configured provider or global
	tracerProvider := cfg.TracerProvider
	if tracerProvider == nil {
		tracerProvider = otel.GetTracerProvider()
	}

	tracer := tracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
	)

	// Get propagators from config or use global
	propagators := cfg.Propagators
	if propagators == nil {
		propagators = otel.GetTextMapPropagator()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Apply request filter if configured
			if cfg.Filter != nil && !cfg.Filter(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract context from headers for distributed tracing
			ctx := propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Generate span name using configured formatter or default
			spanName := cfg.SpanNameFormatter("HTTP "+r.Method, r)

			// Start span with extracted context
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.HTTPRouteKey.String(r.URL.Path),
					semconv.UserAgentOriginalKey.String(r.UserAgent()),
					semconv.URLPathKey.String(r.URL.Path),
					semconv.URLQueryKey.String(r.URL.RawQuery),
				),
			)
			defer span.End()

			// Set additional service attribute
			span.SetAttributes(attribute.String("service.name", service))

			// Create response writer wrapper to capture status code and response size
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default to 200
			}

			// Update request context with span context
			r = r.WithContext(ctx)

			// Call next handler
			next.ServeHTTP(wrapped, r)

			// Set span status based on HTTP status code
			if wrapped.statusCode >= 400 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", wrapped.statusCode))
			} else {
				span.SetStatus(codes.Ok, "")
			}

			// Add response attributes
			span.SetAttributes(
				attribute.Int("http.response.status_code", wrapped.statusCode),
				attribute.Int("http.response.body.size", wrapped.bytesWritten),
			)
		})
	}
}

// FuegoMiddleware is a convenience function that returns a Fuego-compatible middleware
// function that can be used with fuego.Use() directly.
func FuegoMiddleware(service string, opts ...Option) func(http.Handler) http.Handler {
	return Middleware(service, opts...)
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode    int
	bytesWritten  int
	headerWritten bool
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.headerWritten {
		rw.statusCode = statusCode
		rw.headerWritten = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.headerWritten {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(data)
	rw.bytesWritten += n
	return n, err
}

// Flush implements http.Flusher if the underlying ResponseWriter supports it
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack implements http.Hijacker if the underlying ResponseWriter supports it
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("responseWriter does not support hijacking")
}
