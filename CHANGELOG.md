# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of otelfuego middleware
- OpenTelemetry instrumentation for Fuego web framework
- Automatic HTTP request tracing with distributed tracing support
- Configurable request filtering system
- Built-in filters for health checks, path prefixes, and suffixes
- Custom span naming support
- Response metrics capture (status codes and response sizes)
- Error handling with proper span status setting
- Comprehensive test suite with examples

### Features
- Functional options pattern for configuration
- Support for custom tracer providers and propagators
- Framework-specific integration with Fuego middleware patterns
- Compatible with standard http.Handler interface
- Support for http.Flusher and http.Hijacker interfaces

## [0.1.0] - TBD

### Added
- Initial public release