# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-03-02

### Security
- **Fixed timing attack vulnerability** in webhook signature verification — replaced string `==` with `crypto/subtle.ConstantTimeCompare`
- **Streaming SHA256 computation** — avoid allocating large temporary strings in `VerifySignature`
- **Debug logging safety** — replaced `fmt.Printf` with configurable `Logger` interface to prevent accidental secret leakage

### Added
- `Client.Close()` method for releasing resources
- `Logger` interface and `WithLogger()` option for pluggable logging
- `APIError.Unwrap()` method for Go 1.13+ `errors.Is/As` chain support
- Input validation on all batch API methods (empty check + max 40 items)
- API key empty string validation in `New()` (panics early)
- GitHub Actions CI workflow (`.github/workflows/ci.yml`)
- `Makefile` with test, lint, coverage, and CI targets
- `CHANGELOG.md`

### Fixed
- `ParseWebhook` body close ordering — `defer r.Body.Close()` now before `io.ReadAll`
- `rateLimiter.wait()` no longer holds mutex lock during `time.Sleep`
- `rateLimiter.wait()` now accepts `context.Context` and respects cancellation
- `IsAPIError()` now uses `errors.As` for wrapped error chain support

### Changed
- Debug output now goes through `Logger` interface (default: `log.New(os.Stderr, ...)`) instead of `fmt.Printf` to stdout

## [1.0.0] - 2026-02-28

### Added
- Initial release
- Complete 17Track API v2.4 coverage (11 endpoints)
- Zero external dependencies
- Built-in rate limiting (3 req/s)
- Exponential backoff retry for 5xx/429 errors
- Webhook SHA256 signature verification and HTTP handler
- Comprehensive type definitions for all API entities
- Full godoc documentation with examples
