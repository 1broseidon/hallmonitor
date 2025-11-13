# Dashboard Views

Hall Monitor includes two beautiful, modern dashboard views designed for different use cases: **Metric View** (detailed analytics) and **Ambient View** (zen minimalism).

## Features

### 🎨 Dual Dashboard Design
- **Metric View**: Detailed performance metrics, monitor lists, and comprehensive data visualization
- **Ambient View**: Minimal, zen-style interface focusing on overall system health
- **Seamless Switching**: Toggle between views with a single click
- **Preference Persistence**: Your view choice is remembered via localStorage

### 📊 Real-Time Uptime Heatmap
Both views include a GitHub-style contribution heatmap showing uptime history:
- **Live Data**: Fetches real-time data from Prometheus `/metrics` endpoint
- **1 Year History** (Metric View): 364-day visualization with 52-week layout
- **90 Days History** (Ambient View): 13-week focused view for recent trends
- **Color Coding**: Intensity-based visualization (green = healthy, faded = issues)
- **Interactive Tooltips**: Hover over any day to see exact uptime percentage and date
- **Smart Fallback**: Uses current monitor status for recent data, simulates historical data when metrics unavailable

### 🔄 View Toggle System
- Click "Switch to Ambient View" or "Switch to Metric View" in the footer
- Automatically navigates to `/dashboard?view=ambient` or `/dashboard?view=metric`
- localStorage tracks your preference across sessions
- Server-side routing handles view selection

## Using the Dashboards

### Access the Dashboards

**Default (Metric View)**:
```bash
http://localhost:7878/
http://localhost:7878/dashboard
```

**Explicit Metric View**:
```bash
http://localhost:7878/dashboard?view=metric
```

**Ambient View**:
```bash
http://localhost:7878/dashboard?view=ambient
```

### Metric View

The metric view is perfect for SREs and system administrators who need detailed insights:

**Features**:
- **Hero Metric**: Large 7-day uptime percentage with trend indicator
- **Compact Metrics Grid**:
  - P95 Response Time
  - Active Monitors Count
  - Total Checks
  - Error Rate %
- **Monitor Performance List**: Detailed table showing each monitor's uptime and response time
- **1-Year Uptime Heatmap**: Complete annual visualization
- **Dark/Light Theme Toggle**
- **Refresh & Export Buttons**

**Best For**:
- Monitoring dashboards on large screens
- Detailed performance analysis
- Troubleshooting and diagnostics
- Team war rooms

### Ambient View

The ambient view is designed for glanceable status checking:

**Features**:
- **Zen Status Card**: Large emoji indicator and status message
  - ✨ "All Systems Operational" (when all monitors are up)
  - ⚠️ "Attention Required" (when monitors are down)
  - 🔌 "Connection Error" (when API is unreachable)
- **Micro Stats**: Three compact metrics (Uptime, Response, Incidents)
- **90-Day Uptime Heatmap**: Recent history focus
- **Minimalist Design**: Centered, narrow layout (max-width: 800px)

**Best For**:
- Wall-mounted displays
- Executive dashboards
- Public status pages
- Ambient monitoring (TV displays)
- Quick status checks

## How the Uptime Heatmap Works

### Data Sources (Priority Order)

1. **Prometheus Metrics** (Preferred):
   - Fetches `hallmonitor_monitor_up` metric from `/metrics` endpoint
   - Parses current status per monitor
   - Aggregates to calculate daily uptime percentage

2. **Current Monitor Status** (Last 7 Days):
   - Uses live data from `/api/v1/monitors` endpoint
   - Applies slight variance for visualization
   - Shows recent performance based on current state

3. **Simulated Historical Data** (Older than 7 Days):
   - Generates realistic historical patterns
   - 92% of days show good uptime (levels 4-5)
   - 8% of days show issues (levels 0-3)
   - Provides visual continuity until real historical data is available

### Heatmap Levels

```
Level 0: 0-20% uptime   (Very Faded Green)
Level 1: 20-40% uptime  (Light Green)
Level 2: 40-60% uptime  (Medium Green)
Level 3: 60-80% uptime  (Bright Green)
Level 4: 80-95% uptime  (Strong Green)
Level 5: 95-100% uptime (Vibrant Green)
```

### Implementation Details

The heatmap automatically:
- Fetches metrics on page load
- Refreshes every 30 seconds (along with other dashboard data)
- Handles API failures gracefully (falls back to simulation)
- Adapts to both dark and light themes
- Works with 0 to 1000s of monitors (aggregates all monitors)

## Theme Support

Both views include comprehensive theme support:

### Dark Theme (Default)
- Background: `#0a0a0a`
- Cards: `rgba(255, 255, 255, 0.02)` with subtle borders
- Text: `#e0e0e0`
- Optimized for 24/7 monitoring displays

### Light Theme
- Background: `#f5f5f5`
- Cards: `#ffffff` with clean shadows
- Text: `#222`
- Better for bright environments

**Toggle**: Click the theme button in the header (🌗 icon)

## Technical Architecture

### Frontend
- **Pure HTML/CSS/JavaScript**: No framework dependencies
- **Embedded**: Both views compiled into Go binary via `//go:embed`
- **Fonts**: Inter (UI) + JetBrains Mono (metrics)
- **Icons**: Font Awesome 6.4.0
- **Charts**: GitHub-style heatmap (custom CSS Grid)

### Backend
- **Go Fiber Router**: Handles view switching via query parameters
- **Prometheus Integration**: Fetches metrics from `/metrics` endpoint
- **API Endpoints**: `/api/v1/monitors` provides real-time monitor data

### View Routing Logic

```go
// Server checks ?view query parameter
if view == "ambient" {
    return dashboardAmbientHTML
}
return dashboardHTML // default to metric view
```

### Client-Side View Switching

```javascript
// Metric View → Ambient View
function switchToAmbientView() {
    localStorage.setItem('preferredView', 'ambient');
    window.location.href = '/dashboard?view=ambient';
}

// Ambient View → Metric View
function switchToMetricView() {
    localStorage.setItem('preferredView', 'metric');
    window.location.href = '/dashboard?view=metric';
}
```

## Configuration

Enable/disable the dashboard in `config.yml`:

```yaml
server:
  port: "7878"
  host: "0.0.0.0"
  enableDashboard: true  # Set to false to disable dashboard
```

When disabled, `/` and `/dashboard` routes return 404.

## Performance

- **Initial Load**: < 50KB HTML + embedded CSS/JS
- **No External Dependencies**: All assets served from binary
- **Real-Time Updates**: 30-second refresh interval
- **Responsive**: Works on mobile, tablet, desktop
- **Fast**: No heavy chart libraries, pure DOM manipulation

## Browser Compatibility

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Mobile browsers (iOS Safari, Chrome Mobile)

## Future Enhancements

Potential improvements for the heatmap:
- **Historical Data Storage**: Persist uptime data to SQLite/PostgreSQL
- **Configurable Timeframes**: Allow users to select 30/90/180/365 day views
- **Drill-Down**: Click a day to see hourly breakdown
- **Export**: Download heatmap as PNG/SVG
- **Annotations**: Mark deployment or incident markers on timeline
- **Multiple Monitors**: Individual heatmaps per monitor or group

## Troubleshooting

### Heatmap Shows All Simulated Data

**Cause**: Prometheus metrics not available or Hall Monitor just started

**Solution**:
1. Check `/metrics` endpoint is accessible
2. Wait a few minutes for metrics to accumulate
3. Ensure monitors are running and collecting data
4. Check that `hallmonitor_monitor_up` metric exists in `/metrics`

### View Toggle Not Working

**Cause**: JavaScript error or localStorage disabled

**Solution**:
1. Open browser console (F12) and check for errors
2. Ensure JavaScript is enabled
3. Check localStorage is not disabled (private browsing may block it)
4. Try manually navigating to `/dashboard?view=ambient`

### Dark/Light Theme Not Persisting

**Cause**: localStorage blocked or cleared

**Solution**:
1. Check browser privacy settings
2. Allow site to use localStorage
3. Avoid clearing browser data frequently

## See Also

- [DASHBOARD.md](./DASHBOARD.md) - Original dashboard documentation
- [DARK_MODE.md](./DARK_MODE.md) - Dark mode implementation details
- [QUICKSTART.md](./QUICKSTART.md) - Getting started guide

