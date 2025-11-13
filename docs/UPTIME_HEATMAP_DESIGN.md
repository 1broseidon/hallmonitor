# Uptime Heatmap Design Implementation

## Overview

The uptime history heatmap now properly distinguishes between three distinct states:
1. **Past No-Data** - Days when Hall Monitor wasn't running
2. **Actual Uptime Data** - Days with monitoring data from BadgerDB
3. **Future Unknown** - Days that haven't occurred yet

## Visual Design

### Metric View (dashboard.html)

**Color-coded system with visual patterns:**

- **Past No-Data (Striped Gray)**
  - Dark gray with diagonal stripes pattern
  - Border for clear distinction
  - Tooltip: "No data (Hall Monitor not running)"
  
- **Uptime Data (Red → Yellow → Green)**
  - Level 1 (Red): <50% uptime - Critical
  - Level 2 (Orange/Yellow): 50-80% uptime - Poor
  - Level 3 (Yellow): 80-90% uptime - Warning
  - Level 4 (Light Green): 90-98% uptime - Good
  - Level 5 (Green): 98-100% uptime - Excellent
  - Subtle inner border for depth
  - Tooltip: "X.X% uptime" or "X.X% uptime (current)" for today

- **Future Unknown (Light Dotted)**
  - Very light gray background
  - Dotted border for "pending" feel
  - Tooltip: "Not yet occurred"

### Ambient View (dashboard_ambient.html)

**Minimalist design with calm aesthetics:**

- **Past No-Data (Nearly Invisible)**
  - Very dark, subtle gray
  - Minimal border
  - Blends into background

- **Uptime Data (Green Gradient)**
  - Single-hue gradient (green only)
  - Level 1: Very dim (15% opacity) - Low uptime
  - Level 2: Dim (35% opacity)
  - Level 3: Medium (55% opacity)
  - Level 4: Bright (75% opacity)
  - Level 5: Brightest (95% opacity) - Excellent uptime
  - Hover effect adds glow on level 5
  
- **Future Unknown (Faint Blue)**
  - Very light blue tint
  - Subtle border
  - Barely visible, non-intrusive

## Implementation Details

### Key Functions

**`calculateUptimeLevel(uptimeValue)`**
- Converts 0-1 range to 1-5 levels
- Better thresholds for meaningful distinction:
  - <50% → Level 1 (Critical)
  - 50-80% → Level 2 (Poor)
  - 80-90% → Level 3 (Warning)
  - 90-98% → Level 4 (Good)
  - 98-100% → Level 5 (Excellent)

**`generateHeatmap(historyData)`**
- Properly distinguishes between three states
- Checks `isFuture` for dates after today
- Checks `hasHistoricalData` for BadgerDB records
- Checks `isToday` to use live monitor status
- Falls back to `level-past-nodata` for past dates without data

### State Logic Flow

```javascript
if (isFuture) {
    // Show as future (light dotted)
} else if (hasHistoricalData) {
    // Show actual uptime data with color coding
} else if (isToday && monitorsData) {
    // Show current day with live status
} else {
    // Show as past-no-data (striped gray)
}
```

## Data Source

The heatmap pulls data from:
- **BadgerDB Storage**: Historical results via `/api/v1/monitors/:name/history`
- **Live Status**: Current monitor status from `/api/v1/monitors`
- **Aggregation**: Daily uptime percentages calculated from stored check results

## User Experience

### Tooltips
Each cell shows detailed information on hover:
- Past no-data: "Jan 15, 2025: No data (Hall Monitor not running)"
- Uptime data: "Jan 20, 2025: 99.8% uptime"
- Current day: "Jan 22, 2025: 100.0% uptime (current)"
- Future: "Jan 25, 2025: Not yet occurred"

### Accessibility
- Color + Pattern (stripes/borders) helps colorblind users
- Hover tooltips provide textual information
- Legend shows gradient scale with labels

### Time Ranges
- **Metric View**: 7d, 30d, 90d (defaults to 90d)
- **Ambient View**: 1d (hourly), 7d, 30d (defaults to 30d)
- Settings persist in localStorage per view

## Theme Support

Both light and dark themes are fully supported with appropriate color adjustments:
- Dark theme: Higher contrast, vibrant colors
- Light theme: Softer colors, reduced intensity

## Future Enhancements

Potential improvements:
- Hourly granularity for recent data
- Click to view detailed check history
- Animation on data updates
- Multiple monitor aggregation view
- Export heatmap as image

