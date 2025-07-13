# Fuego OpenTelemetry Middleware

[![CI](https://github.com/pdrvsky/otelfuego/actions/workflows/ci.yml/badge.svg)](https://github.com/pdrvsky/otelfuego/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/pdrvsky/otelfuego.svg)](https://pkg.go.dev/github.com/pdrvsky/otelfuego)
[![Go Report Card](https://goreportcard.com/badge/github.com/pdrvsky/otelfuego)](https://goreportcard.com/report/github.com/pdrvsky/otelfuego)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![codecov](https://codecov.io/gh/pdrvsky/otelfuego/branch/main/graph/badge.svg)](https://codecov.io/gh/pdrvsky/otelfuego)

A specialized OpenTelemetry middleware for the [Fuego](https://github.com/go-fuego/fuego) Go web framework that provides automatic HTTP request tracing with distributed tracing support.

## Features

- ✅ **Automatic HTTP Instrumentation**: Traces all HTTP requests with minimal setup
- ✅ **Distributed Tracing Support**: Extracts and propagates trace context for microservices
- ✅ **Flexible Configuration**: Functional options pattern for customization
- ✅ **Request Filtering**: Skip tracing for health checks and other endpoints
- ✅ **Custom Span Naming**: Configure how spans are named
- ✅ **Response Metrics**: Captures HTTP status codes and response sizes
- ✅ **Error Handling**: Proper span status setting based on HTTP status codes
- ✅ **Framework Integration**: Designed specifically for Fuego's middleware patterns

## Installation

```bash
go get github.com/pdrvsky/otelfuego
```

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/go-fuego/fuego"
    "github.com/pdrvsky/otelfuego/otelfuego"
)

func main() {
    // Initialize your OpenTelemetry tracer provider (not shown)
    // setupTracing()

    server := fuego.NewServer()
    
    // Add OpenTelemetry middleware
    server.Use(otelfuego.Middleware("my-service"))
    
    // Define your routes
    fuego.Get(server, "/hello", func(c fuego.ContextNoBody) (string, error) {
        return "Hello World", nil
    })
    
    server.Run()
}
```

### Advanced Configuration

```go
server.Use(otelfuego.Middleware("my-service",
    // Skip health check endpoints
    otelfuego.WithFilter(otelfuego.HealthCheckFilter()),
    
    // Custom span naming
    otelfuego.WithSpanNameFormatter(func(operation string, r *http.Request) string {
        return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
    }),
    
    // Custom tracer provider
    otelfuego.WithTracerProvider(customTracerProvider),
    
    // Custom propagators
    otelfuego.WithPropagators(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    )),
))
```

## Configuration Options

### WithFilter

Control which requests should be traced:

```go
// Skip health check endpoints
otelfuego.WithFilter(otelfuego.HealthCheckFilter())

// Skip paths with specific prefix
otelfuego.WithFilter(otelfuego.PathPrefixFilter("/internal"))

// Skip paths with specific suffix
otelfuego.WithFilter(otelfuego.PathSuffixFilter(".js"))

// Combine multiple filters
otelfuego.WithFilter(otelfuego.CombineFilters(
    otelfuego.HealthCheckFilter(),
    otelfuego.PathPrefixFilter("/static"),
    func(req *http.Request) bool {
        return req.Method != "OPTIONS"
    },
))

// Custom filter
otelfuego.WithFilter(func(req *http.Request) bool {
    return !strings.Contains(req.URL.Path, "/private")
})
```

### WithSpanNameFormatter

Customize how spans are named:

```go
otelfuego.WithSpanNameFormatter(func(operation string, r *http.Request) string {
    // Include query parameters in span name
    if r.URL.RawQuery != "" {
        return fmt.Sprintf("%s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
    }
    return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
})
```

### WithTracerProvider

Use a custom tracer provider:

```go
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
    sdktrace.WithSampler(sdktrace.AlwaysSample()),
)

server.Use(otelfuego.Middleware("my-service",
    otelfuego.WithTracerProvider(tp),
))
```

### WithPropagators

Configure context propagation for distributed tracing:

```go
propagator := propagation.NewCompositeTextMapPropagator(
    propagation.TraceContext{},
    propagation.Baggage{},
    b3.New(),
)

server.Use(otelfuego.Middleware("my-service",
    otelfuego.WithPropagators(propagator),
))
```

## Built-in Filters

### HealthCheckFilter

Excludes common health check endpoints:
- `/health`
- `/healthz`
- `/ping`
- `/ready`
- `/live`
- `/metrics`

```go
otelfuego.WithFilter(otelfuego.HealthCheckFilter())
```

### PathPrefixFilter

Excludes paths starting with a specific prefix:

```go
otelfuego.WithFilter(otelfuego.PathPrefixFilter("/static"))
```

### PathSuffixFilter

Excludes paths ending with a specific suffix:

```go
otelfuego.WithFilter(otelfuego.PathSuffixFilter(".ico"))
```

### CombineFilters

Combines multiple filters with AND logic:

```go
otelfuego.WithFilter(otelfuego.CombineFilters(
    otelfuego.HealthCheckFilter(),
    otelfuego.PathPrefixFilter("/static"),
    customFilter,
))
```

## Complete Example with OpenTelemetry Setup

```go
package main

import (
    "context"
    "log"
    "os"
    
    "github.com/go-fuego/fuego"
    "github.com/pdrvsky/otelfuego/otelfuego"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

func setupTracing() func() {
    ctx := context.Background()
    
    // Create OTLP HTTP exporter
    exporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint("http://localhost:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        log.Fatal("Failed to create exporter:", err)
    }
    
    // Create resource
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String(os.Getenv("SERVICE_NAME")),
            semconv.ServiceVersionKey.String("1.0.0"),
        ),
    )
    if err != nil {
        log.Fatal("Failed to create resource:", err)
    }
    
    // Create tracer provider
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
    )
    
    // Set global providers
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))
    
    return func() {
        tp.Shutdown(ctx)
    }
}

func main() {
    // Setup OpenTelemetry
    shutdown := setupTracing()
    defer shutdown()
    
    // Create Fuego server
    server := fuego.NewServer()
    
    // Add OpenTelemetry middleware
    server.Use(otelfuego.Middleware("my-fuego-service",
        otelfuego.WithFilter(otelfuego.HealthCheckFilter()),
    ))
    
    // Define routes
    fuego.Get(server, "/", func(c fuego.ContextNoBody) (string, error) {
        return "Hello World", nil
    })
    
    fuego.Post(server, "/heavy-work", func(c fuego.ContextWithBody[map[string]string]) (map[string]interface{}, error) {
        body := c.Body()
        
        // This will be automatically traced by the middleware
        // Add custom spans if needed:
        // ctx, span := otel.Tracer("my-tracer").Start(c.Context(), "custom-operation")
        // defer span.End()
        
        return map[string]interface{}{
            "message": "Work completed",
            "input":   body,
        }, nil
    })
    
    // Health check endpoint (will be filtered out)
    fuego.Get(server, "/health", func(c fuego.ContextNoBody) (string, error) {
        return "OK", nil
    })
    
    server.Run()
}
```

## Integration with SigNoz

This middleware works seamlessly with [SigNoz](https://signoz.io/) and other OpenTelemetry-compatible observability platforms:

1. **Configure the exporter** to point to your SigNoz collector endpoint
2. **Set environment variables**:
   ```bash
   export SERVICE_NAME=my-fuego-service
   export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
   ```
3. **Start your service** and see traces in the SigNoz UI

## Comparison with Generic HTTP Middleware

| Feature | `otelfuego` | `otelhttp.NewHandler()` |
|---------|-------------|-------------------------|
| Framework Integration | ✅ Fuego-specific | ❌ Generic HTTP |
| Configuration Options | ✅ Rich functional options | ❌ Limited |
| Request Filtering | ✅ Built-in filters | ❌ Manual implementation |
| Custom Span Naming | ✅ Configurable formatters | ❌ Fixed format |
| Response Metrics | ✅ Status code + size | ✅ Basic |
| Error Handling | ✅ HTTP status-aware | ✅ Basic |
| Documentation | ✅ Framework-specific examples | ❌ Generic examples |

## Migration from otelhttp

### Before (using otelhttp)

```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

server.Use(func(next http.Handler) http.Handler {
    return otelhttp.NewHandler(next, "my-service")
})
```

### After (using otelfuego)

```go
import "github.com/pdrvsky/otelfuego/otelfuego"

server.Use(otelfuego.Middleware("my-service"))
```

### With filtering (not easily possible with otelhttp)

```go
server.Use(otelfuego.Middleware("my-service",
    otelfuego.WithFilter(otelfuego.HealthCheckFilter()),
))
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Related Projects

- [Fuego](https://github.com/go-fuego/fuego) - The Go web framework this middleware is designed for
- [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go) - OpenTelemetry Go SDK
- [SigNoz](https://github.com/SigNoz/signoz) - OpenTelemetry-native observability platform