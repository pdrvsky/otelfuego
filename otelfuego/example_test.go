package otelfuego_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pdrvsky/otelfuego/otelfuego"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestMiddleware_BasicUsage(t *testing.T) {
	// Setup in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	// Setup middleware
	middleware := otelfuego.Middleware("test-service",
		otelfuego.WithTracerProvider(tp),
	)

	// Create test handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello World"))
	}))

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify span was created
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name != "GET /test" {
		t.Errorf("Expected span name 'GET /test', got '%s'", span.Name)
	}
}

func TestMiddleware_WithFilter(t *testing.T) {
	// Setup in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	// Setup middleware with health check filter
	middleware := otelfuego.Middleware("test-service",
		otelfuego.WithTracerProvider(tp),
		otelfuego.WithFilter(func(req *http.Request) bool {
			return req.URL.Path != "/health"
		}),
	)

	// Create test handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Test filtered request (should not create span)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify no spans created for filtered request
	spans := exporter.GetSpans()
	if len(spans) != 0 {
		t.Errorf("Expected 0 spans for filtered request, got %d", len(spans))
	}

	// Test non-filtered request (should create span)
	req = httptest.NewRequest("GET", "/api/users", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify span was created for non-filtered request
	spans = exporter.GetSpans()
	if len(spans) != 1 {
		t.Errorf("Expected 1 span for non-filtered request, got %d", len(spans))
	}
}

func TestMiddleware_WithCustomSpanNameFormatter(t *testing.T) {
	// Setup in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	// Setup middleware with custom span name formatter
	middleware := otelfuego.Middleware("test-service",
		otelfuego.WithTracerProvider(tp),
		otelfuego.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return "custom-" + r.Method + "-" + strings.TrimPrefix(r.URL.Path, "/")
		}),
	)

	// Create test handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create test request
	req := httptest.NewRequest("POST", "/api/users", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify custom span name
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Errorf("Expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	expectedName := "custom-POST-api/users"
	if span.Name != expectedName {
		t.Errorf("Expected span name '%s', got '%s'", expectedName, span.Name)
	}
}

func TestMiddleware_DistributedTracing(t *testing.T) {
	// Setup in-memory span exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	defer tp.Shutdown(context.Background())

	// Setup propagators
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	// Setup middleware
	middleware := otelfuego.Middleware("test-service",
		otelfuego.WithTracerProvider(tp),
		otelfuego.WithPropagators(propagator),
	)

	// Create test handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create parent span context
	tracer := tp.Tracer("test")
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent-span")
	defer parentSpan.End()

	// Create request with trace context headers
	req := httptest.NewRequest("GET", "/test", nil)
	propagator.Inject(parentCtx, propagation.HeaderCarrier(req.Header))

	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify spans
	spans := exporter.GetSpans()
	if len(spans) != 2 { // parent + child span
		t.Errorf("Expected 2 spans, got %d", len(spans))
	}

	// Find child span (should have parent trace ID)
	var childSpan *tracetest.SpanStub
	for _, span := range spans {
		if span.Name == "GET /test" {
			childSpan = &span
			break
		}
	}

	if childSpan == nil {
		t.Error("Child span not found")
		return
	}

	if !childSpan.SpanContext.IsValid() {
		t.Error("Child span context is invalid")
	}

	if childSpan.SpanContext.TraceID() != parentSpan.SpanContext().TraceID() {
		t.Error("Child span does not have same trace ID as parent")
	}
}

func ExampleMiddleware() {
	// Basic usage with default configuration
	middleware := otelfuego.Middleware("my-service")

	// Create a simple handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello World"))
	}))

	// Use with your HTTP server
	http.ListenAndServe(":8080", handler)
}

func ExampleMiddleware_withOptions() {
	// Advanced usage with custom configuration
	middleware := otelfuego.Middleware("my-service",
		// Skip health check endpoints
		otelfuego.WithFilter(otelfuego.HealthCheckFilter()),

		// Custom span naming
		otelfuego.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)

	// Create a simple handler
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello World"))
	}))

	// Use with your HTTP server
	http.ListenAndServe(":8080", handler)
}

func ExampleHealthCheckFilter() {
	// Use the built-in health check filter
	middleware := otelfuego.Middleware("my-service",
		otelfuego.WithFilter(otelfuego.HealthCheckFilter()),
	)

	// This will skip tracing for /health, /healthz, /ping, /ready, /live, /metrics
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	http.ListenAndServe(":8080", handler)
}

func ExampleCombineFilters() {
	// Combine multiple filters
	middleware := otelfuego.Middleware("my-service",
		otelfuego.WithFilter(otelfuego.CombineFilters(
			otelfuego.HealthCheckFilter(),
			otelfuego.PathPrefixFilter("/internal"),
			func(req *http.Request) bool {
				// Custom filter: skip OPTIONS requests
				return req.Method != "OPTIONS"
			},
		)),
	)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	http.ListenAndServe(":8080", handler)
}
