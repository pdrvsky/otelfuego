package otelfuego

import (
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// config holds the configuration for the OpenTelemetry middleware
type config struct {
	TracerProvider    trace.TracerProvider
	Propagators       propagation.TextMapPropagator
	Filter            Filter
	SpanNameFormatter SpanNameFormatter
}

// Option is a function that configures the middleware
type Option interface {
	apply(*config)
}

// optionFunc wraps a function to implement the Option interface
type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// Filter is a function that determines whether a request should be traced
type Filter func(*http.Request) bool

// SpanNameFormatter is a function that formats the span name based on the operation and request
type SpanNameFormatter func(operation string, r *http.Request) string

// newConfig creates a new config with default values and applies the given options
func newConfig(opts ...Option) *config {
	c := &config{
		SpanNameFormatter: defaultSpanNameFormatter,
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c
}

// defaultSpanNameFormatter is the default span name formatter
func defaultSpanNameFormatter(operation string, r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}

// WithTracerProvider configures the middleware to use a specific tracer provider
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(c *config) {
		c.TracerProvider = provider
	})
}

// WithPropagators configures the middleware to use specific propagators for context propagation
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(c *config) {
		c.Propagators = propagators
	})
}

// WithFilter configures the middleware to use a filter function to determine which requests to trace
// The filter function should return true for requests that should be traced, false otherwise.
//
// Example:
//
//	WithFilter(func(req *http.Request) bool {
//	    return !strings.Contains(req.URL.Path, "/health")
//	})
func WithFilter(filter Filter) Option {
	return optionFunc(func(c *config) {
		c.Filter = filter
	})
}

// WithSpanNameFormatter configures the middleware to use a custom span name formatter
//
// Example:
//
//	WithSpanNameFormatter(func(operation string, req *http.Request) string {
//	    return fmt.Sprintf("%s %s", req.Method, req.URL.Path)
//	})
func WithSpanNameFormatter(formatter SpanNameFormatter) Option {
	return optionFunc(func(c *config) {
		c.SpanNameFormatter = formatter
	})
}

// Common filter functions for convenience

// HealthCheckFilter returns a filter that excludes common health check endpoints
func HealthCheckFilter() Filter {
	return func(req *http.Request) bool {
		path := req.URL.Path
		return path != "/health" &&
			path != "/healthz" &&
			path != "/ping" &&
			path != "/ready" &&
			path != "/live" &&
			path != "/metrics"
	}
}

// PathPrefixFilter returns a filter that excludes paths with the given prefix
func PathPrefixFilter(prefix string) Filter {
	return func(req *http.Request) bool {
		return !strings.HasPrefix(req.URL.Path, prefix)
	}
}

// PathSuffixFilter returns a filter that excludes paths with the given suffix
func PathSuffixFilter(suffix string) Filter {
	return func(req *http.Request) bool {
		return !strings.HasSuffix(req.URL.Path, suffix)
	}
}

// CombineFilters combines multiple filters with AND logic (all must return true)
func CombineFilters(filters ...Filter) Filter {
	return func(req *http.Request) bool {
		for _, filter := range filters {
			if !filter(req) {
				return false
			}
		}
		return true
	}
}
