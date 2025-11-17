# Changelog

All notable changes to Hall Monitor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- PostgreSQL/TimescaleDB storage backend support
- InfluxDB storage backend support
- PostgresStore implementation with full ResultStore interface
- InfluxDBStore implementation with Flux query support
- Automatic schema initialization for PostgreSQL
- TimescaleDB hypertable support (optional)
- Connection pooling for PostgreSQL (configurable)
- Health check endpoints for storage backends
- Retention cleanup for PostgreSQL (daily background task)
- Integration tests for PostgreSQL backend
- Integration tests for InfluxDB backend
- Docker Compose test infrastructure (docker-compose.test.yml)
- PostgreSQL configuration example (examples/config-postgres.yml)
- InfluxDB configuration example (examples/config-influxdb.yml)
- Comprehensive storage backends documentation (docs/storage-backends.md)
- Web UI support for PostgreSQL configuration
- Web UI support for InfluxDB configuration
- Four storage backend options in config UI (BadgerDB, PostgreSQL, InfluxDB, None)

### Changed
- Updated storage factory to support postgres and influxdb backends
- Extended StorageConfig with PostgresConfig and InfluxDBConfig
- Added default configurations for new backends
- Enhanced config.html with grid layout for 4 backend options
- Updated error messages to include new backend options

### Dependencies
- Added github.com/jackc/pgx/v5 for PostgreSQL connectivity
- Added github.com/jackc/pgx/v5/pgxpool for connection pooling
- Added github.com/influxdata/influxdb-client-go/v2 for InfluxDB support

### Documentation
- Storage backends comparison matrix
- Backend selection guide
- Security best practices for database credentials
- Performance recommendations for each backend
- Migration guide (planned for future implementation)
- Troubleshooting section for common issues

## [0.4.0] - 2025-11-16

### Added
- Storage backend abstraction layer with pluggable architecture
- `ResultStore` interface for all storage backends to implement
- NoOp storage backend for metrics-only deployments
- Factory pattern for storage backend instantiation
- Backend capability detection system (`BackendCapabilities`)
- Web-based storage configuration UI in `/config` page
- Visual backend selection (BadgerDB vs Metrics Only)
- Storage settings panel with path, retention, and aggregation controls
- Comprehensive factory tests (8 new test cases, 32 total storage tests)
- Production-ready error handling with nil validation
- HTTP 501 responses for unsupported storage operations
- Helpful error messages guiding users to enable required backends

### Changed
- BadgerDB storage now implements `ResultStore` interface
- Storage configuration structure with nested `badger` section
- Server initialization to use factory pattern instead of direct BadgerStore creation
- Config API endpoints to expose new storage structure
- Example configuration file with new storage backend format
- API handlers to check storage capabilities before operations

### Fixed
- Added nil logger validation in storage factory
- Added `server` binary to .gitignore to prevent accidental commits

### Migration Notes
- Existing configurations continue to work (backward compatible)
- Old format: `storage.enabled: true` automatically maps to `backend: "badger"`
- New format recommended: `storage.backend: "badger"` with nested settings
- No breaking changes - seamless upgrade path

## [0.3.0] - 2025-11-16

### Added
- Comprehensive test suite increasing coverage from 58% to 70.7%
- Test helper pattern in `scheduler` package for safe test data injection
- DNS monitor validation and integration tests
- Logger method tests (Debug, Info, Warn, Error with formatting)
- MonitorManager tests (LoadMonitors, GetMonitors, GetGroups)
- API handler tests for dashboard, history, and uptime endpoints
- Testing guidelines documentation in `claude.md`
- Shared HTML template architecture with reusable components
- Template partials for header and header styles
- Zen-like ambient dashboard view with minimalist design
- Build tags for separating unit and integration tests
- Auto-refresh functionality using htmx for dashboard data
- Alpine.js-based client-side reactivity

### Changed
- Migrated templates from embedded HTML to `internal/api/templates/` directory
- Refactored dashboard to use shared header component across all views
- Migrated theme toggle to Alpine.js (from vanilla JavaScript)
- Migrated search filter to Alpine.js
- Migrated row expansion to Alpine.js with x-collapse directive
- Migrated heatmap time range selector to Alpine.js
- Redesigned ambient dashboard with large uptime display and minimal stats
- Updated dashboard hero section with cleaner layout

### Fixed
- Removed unsafe `reflect` and `unsafe` package usage from tests
- Added cache-control headers to prevent stale JavaScript issues
- Added null checks to DOM element access in updateUI function
- Fixed Alpine.js collapse plugin initialization
- Removed CSS `display:none` conflicts with Alpine's `x-show` directive
- Fixed ambient dashboard view switching

### Removed
- Legacy embedded HTML dashboard files
- Reflection-based test helpers in favor of explicit test utilities

## [0.2.0] - 2025-11-07

### Added
- BadgerDB-backed persistent storage for historical uptime tracking
- Automatic hourly and daily data aggregation
- Historical data API endpoints (`/api/v1/monitors/:name/history`, `/api/v1/monitors/:name/uptime`)
- Real historical data in dashboards with heatmap visualization
- Configurable retention periods (default 30 days)
- Multi-period uptime statistics (24h, 7d, 30d)
- Advanced dashboard with historical analytics
- Full test coverage for storage layer
- Comprehensive storage documentation

### Changed
- Storage layer now supports optional persistent backend
- Dashboard displays real historical data instead of simulated data

### Fixed
- Storage and scheduler coordination issues

## [0.1.0] - 2025-11-05

### Added
- Initial release of Hall Monitor
- HTTP, HTTPS, TCP, Ping, and DNS monitor types
- Configurable check intervals and timeouts
- Prometheus metrics export
- Built-in web dashboard
- Grafana JSON API endpoints (placeholder implementation for future development)
- Docker support with multi-architecture builds (amd64, arm64)
- Comprehensive documentation
- GitHub Actions CI/CD workflows

### Changed
- Default port changed from 8080 to 7878
- ReadBufferSize increased to 16KB for mobile browser compatibility

### Fixed
- Request Header Fields Too Large error for remote mobile access

[unreleased]: https://github.com/1broseidon/hallmonitor/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/1broseidon/hallmonitor/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/1broseidon/hallmonitor/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/1broseidon/hallmonitor/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/1broseidon/hallmonitor/releases/tag/v0.1.0
