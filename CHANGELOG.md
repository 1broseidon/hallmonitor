# Changelog

All notable changes to Hall Monitor will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[unreleased]: https://github.com/1broseidon/hallmonitor/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/1broseidon/hallmonitor/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/1broseidon/hallmonitor/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/1broseidon/hallmonitor/releases/tag/v0.1.0
