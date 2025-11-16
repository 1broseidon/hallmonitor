# htmx + Alpine.js Migration Plan
## Hall Monitor Dashboard - Vanilla JS to Modern Stack

---

## 🎯 Migration Goals

1. **Add htmx + Alpine WITHOUT breaking current functionality**
2. **Progressive migration** - test each step before proceeding
3. **Keep it simple** - No templ, use stdlib Go templates for partials
4. **Maintain beautiful S-tier design** - Zero design changes
5. **Reduce JavaScript complexity** - Move logic to appropriate layer

---

## 📊 Current JavaScript Inventory (29 functions)

### **State Management (4 functions)**
- `initTheme()` - Load saved theme → **Alpine**
- `toggleTheme()` - Toggle dark/light → **Alpine**
- `initSettings()` - Load preferences → **Alpine**
- `checkViewPreference()` - Auto-redirect to preferred view → **Keep JS**

### **Data Loading (4 functions)**
- `loadData()` - Fetch monitor JSON → **htmx**
- `loadUptimeHistory()` - Fetch history → **htmx**
- `loadUptimeStats()` - Fetch stats → **htmx**
- `refreshData()` - Manual refresh → **htmx**

### **UI Rendering (5 functions)**
- `updateUI(data)` - Update all dashboard elements → **Go templates + htmx**
- `renderTable()` - Render monitor table → **Go templates + htmx**
- `renderDetailBlocks()` - Render row details → **Go templates + htmx**
- `updateMonitorList()` - Update monitor list → **htmx**
- `generateHeatmap()` - Render heatmap grid → **Keep JS initially**

### **User Interactions (4 functions)**
- `toggleRow()` - Expand/collapse table rows → **Alpine**
- `handleSearch()` - Filter monitors → **Alpine**
- `setTimeRange()` - Change heatmap range → **Alpine**
- `switchToAmbientView()` - View switching → **Keep JS**

### **Utilities (12 functions)**
- `parseDuration()` - Parse duration strings → **Keep JS**
- `formatTimeAgo()` - Format timestamps → **Keep JS**
- `formatDate()` - Format dates → **Keep JS**
- `formatNumber()` - Format numbers → **Keep JS**
- `escapeHtml()` - Escape HTML → **Keep JS**
- `calculateUptimeLevel()` - Calculate heatmap level → **Keep JS**
- `calculateOptimalGridLayout()` - Grid layout math → **Keep JS**
- `parseHistoricalResults()` - Parse history data → **Keep JS**
- `updateTimeRangeButtons()` - Update button states → **Alpine**
- `handleResize()` - Debounced resize handler → **Keep JS**
- `exportToGrafana()` - Export dashboard → **Keep JS**
- `_generateHeatmapCells()` - Generate heatmap cells → **Keep JS**

---

## 🚀 Migration Steps (6 Phases)

### **Phase 0: Preparation** ✅ DONE
- [x] Remove templ migration
- [x] Restore vanilla HTML/JS
- [x] Verify dashboard works
- [x] Create migration plan

---

### **Phase 1: Add htmx + Alpine (Non-Breaking)**
**Goal:** Add libraries without changing any functionality

**Steps:**
1. Add htmx CDN to `<head>` (v1.9.10)
2. Add Alpine.js CDN to `<head>` (v3.13.3)
3. Test dashboard still works exactly as before
4. Commit: "chore: add htmx and Alpine.js CDNs"

**Files Modified:**
- `internal/api/dashboard.html` (add CDN links)
- `internal/api/dashboard_ambient.html` (add CDN links)

**Validation:**
- Dashboard loads without errors
- All JavaScript still works
- No console errors
- Theme toggle works
- Table expand/collapse works

---

### **Phase 2: Convert Theme Toggle to Alpine**
**Goal:** Replace theme toggle JavaScript with Alpine

**Current Implementation:**
```javascript
function toggleTheme() {
    const html = document.documentElement;
    const newTheme = html.dataset.theme === 'dark' ? 'light' : 'dark';
    html.dataset.theme = newTheme;
    localStorage.setItem('hallmonitor_theme', newTheme);
}
```

**New Alpine Implementation:**
```html
<div x-data="themeManager()">
  <button @click="toggle()" class="header-btn">
    <i class="fas fa-circle-half-stroke"></i>
  </button>
</div>

<script>
function themeManager() {
  return {
    init() {
      const saved = localStorage.getItem('hallmonitor_theme') || 'dark';
      document.documentElement.dataset.theme = saved;
    },
    toggle() {
      const html = document.documentElement;
      const newTheme = html.dataset.theme === 'dark' ? 'light' : 'dark';
      html.dataset.theme = newTheme;
      localStorage.setItem('hallmonitor_theme', newTheme);
    }
  }
}
</script>
```

**Steps:**
1. Add Alpine `x-data` to header
2. Replace `onclick="toggleTheme()"` with `@click="toggle()"`
3. Test theme toggle works
4. Remove old `toggleTheme()` and `initTheme()` functions
5. Commit: "refactor: migrate theme toggle to Alpine.js"

**Files Modified:**
- `internal/api/dashboard.html`
- `internal/api/dashboard_ambient.html`

**Validation:**
- Theme toggle works
- Theme persists on refresh
- No console errors

---

### **Phase 3: Convert Table Row Expansion to Alpine**
**Goal:** Replace `toggleRow()` with Alpine `x-collapse`

**Current Implementation:**
```javascript
function toggleRow(rowElement) {
    const detailRow = rowElement.nextElementSibling;
    // Close other expanded rows
    document.querySelectorAll('.main-row.expanded').forEach(row => {
        if (row !== rowElement) {
            row.classList.remove('expanded');
            const otherDetail = row.nextElementSibling;
            if (otherDetail && otherDetail.classList.contains('detail-row')) {
                otherDetail.classList.remove('active');
            }
        }
    });
    // Toggle current row
    rowElement.classList.toggle('expanded');
    if (detailRow && detailRow.classList.contains('detail-row')) {
        detailRow.classList.toggle('active');
    }
}
```

**New Alpine Implementation:**
```html
<tbody x-data="{ expandedRow: null }">
  <tr class="main-row"
      :class="{ 'expanded': expandedRow === 'monitor-1' }"
      @click="expandedRow = expandedRow === 'monitor-1' ? null : 'monitor-1'">
    <td>Monitor Name</td>
  </tr>
  <tr class="detail-row" x-show="expandedRow === 'monitor-1'" x-collapse>
    <td colspan="6">
      <div class="detail-content">Details here</div>
    </td>
  </tr>
</tbody>
```

**Steps:**
1. Add `x-data="{ expandedRow: null }"` to `<tbody>`
2. Replace `onclick="toggleRow(this)"` with `@click` handler
3. Add `:class` binding for expanded state
4. Add `x-show` and `x-collapse` to detail rows
5. Test expand/collapse works smoothly
6. Remove old `toggleRow()` function
7. Commit: "refactor: migrate row expansion to Alpine.js"

**Files Modified:**
- `internal/api/dashboard.html` (in `renderTable()` function)

**Validation:**
- Click row to expand
- Click again to collapse
- Expanding one row closes others
- Smooth animation with `x-collapse`

---

### **Phase 4: Convert Search Filter to Alpine**
**Goal:** Replace search JavaScript with Alpine reactivity

**Current Implementation:**
```javascript
let searchQuery = "";
function handleSearch() {
    searchQuery = document.getElementById('searchInput').value.toLowerCase();
    const allMonitors = monitorsData?.monitors || [];
    if (searchQuery === "") {
        filteredMonitors = [...allMonitors];
    } else {
        filteredMonitors = allMonitors.filter(monitor =>
            monitor.name.toLowerCase().includes(searchQuery) ||
            monitor.type.toLowerCase().includes(searchQuery) ||
            // ... more conditions
        );
    }
    displayedCount = Math.min(50, filteredMonitors.length);
    renderTable();
}
```

**New Alpine Implementation:**
```html
<div x-data="monitorSearch()">
  <input type="text"
         x-model="query"
         @input.debounce.300ms="search()"
         placeholder="Search monitors...">

  <template x-for="monitor in filteredMonitors" :key="monitor.name">
    <tr>...</tr>
  </template>
</div>
```

**Steps:**
1. Add Alpine `x-data` for search state
2. Bind input to `x-model="query"`
3. Add debounced `@input` handler
4. Update `renderTable()` to use Alpine filtered data
5. Test search filtering works
6. Remove old `handleSearch()` function
7. Commit: "refactor: migrate search filter to Alpine.js"

**Files Modified:**
- `internal/api/dashboard.html`

**Validation:**
- Type in search box
- Table filters after 300ms
- Clear search shows all monitors
- Search is case-insensitive

---

### **Phase 5: Add htmx Auto-Refresh for Data**
**Goal:** Replace manual `setInterval` with htmx polling

**Current Implementation:**
```javascript
document.addEventListener('DOMContentLoaded', async () => {
    await loadData();
    await loadUptimeHistory();
    setInterval(async () => {
        await loadData();
        await loadUptimeHistory();
    }, 30000); // Refresh every 30s
});

async function loadData() {
    const response = await fetch(`${API_ENDPOINT}/monitors`);
    const data = await response.json();
    monitorsData = data;
    updateUI(data);
}
```

**New htmx Implementation:**

**Option A: Full Page Swap (Simplest)**
```html
<body hx-get="/dashboard"
      hx-trigger="every 30s"
      hx-swap="outerHTML"
      hx-select="body">
  <!-- Dashboard content -->
</body>
```

**Option B: Partial Swaps (More Granular)**
```html
<!-- Hero Section -->
<div id="hero-section"
     hx-get="/api/v1/partials/hero"
     hx-trigger="every 30s"
     hx-swap="outerHTML">
  <!-- Hero content -->
</div>

<!-- Monitor Table -->
<div id="monitor-table"
     hx-get="/api/v1/partials/monitors"
     hx-trigger="every 30s"
     hx-swap="outerHTML">
  <!-- Table content -->
</div>

<!-- Heatmap -->
<div id="heatmap"
     hx-get="/api/v1/partials/heatmap"
     hx-trigger="every 30s"
     hx-swap="outerHTML">
  <!-- Heatmap content -->
</div>
```

**We'll use Option B** for better UX (no full page flash)

**Backend Changes Needed:**
```go
// internal/api/partials.go (new file)
package api

import "html/template"

//go:embed partials/*.html
var partialsFS embed.FS

var partials = template.Must(template.ParseFS(partialsFS, "partials/*.html"))

func (s *Server) heroPartialHandler(c *fiber.Ctx) error {
    data := s.getHeroData()
    return partials.ExecuteTemplate(c.Response().BodyWriter(), "hero.html", data)
}

func (s *Server) monitorsPartialHandler(c *fiber.Ctx) error {
    data := s.getMonitorsData()
    return partials.ExecuteTemplate(c.Response().BodyWriter(), "monitors.html", data)
}
```

**Directory Structure:**
```
internal/api/
  partials/
    hero.html       # Just the hero section HTML
    monitors.html   # Just the monitor table HTML
    heatmap.html    # Just the heatmap HTML
```

**Steps:**
1. Create `internal/api/partials/` directory
2. Extract hero, monitors, heatmap HTML into separate partial files
3. Create Go template handlers for each partial
4. Add routes in `server.go`
5. Add `hx-get`, `hx-trigger`, `hx-swap` to dashboard.html
6. Test auto-refresh works every 30s
7. Remove `setInterval` and `loadData()` calls
8. Commit: "refactor: add htmx auto-refresh for dashboard sections"

**Files Created:**
- `internal/api/partials/hero.html`
- `internal/api/partials/monitors.html`
- `internal/api/partials/heatmap.html`
- `internal/api/partials.go`

**Files Modified:**
- `internal/api/dashboard.html`
- `internal/api/server.go`

**Validation:**
- Dashboard loads normally
- After 30s, sections refresh automatically
- No page flash or jumpiness
- Expanded rows stay expanded during refresh
- Search filter persists during refresh

---

### **Phase 6: Convert Heatmap Time Range to Alpine**
**Goal:** Replace time range buttons with Alpine state

**Current Implementation:**
```javascript
let currentTimeRange = 90;
function setTimeRange(days) {
    currentTimeRange = days;
    localStorage.setItem('hallmonitor_timeRange_metric', days.toString());
    updateTimeRangeButtons();
    generateHeatmap(uptimeHistory);
}
```

**New Alpine Implementation:**
```html
<div x-data="heatmapManager()">
  <div class="time-range-toggle">
    <button @click="setRange(7)"
            :class="{ 'active': range === 7 }">7d</button>
    <button @click="setRange(30)"
            :class="{ 'active': range === 30 }">30d</button>
    <button @click="setRange(90)"
            :class="{ 'active': range === 90 }">90d</button>
  </div>

  <div hx-get="/api/v1/partials/heatmap"
       hx-trigger="rangeChanged from:window"
       hx-vals="js:{ range: Alpine.store('heatmap').range }">
    <!-- Heatmap grid -->
  </div>
</div>

<script>
function heatmapManager() {
  return {
    range: 90,
    init() {
      const saved = localStorage.getItem('hallmonitor_timeRange_metric');
      this.range = saved ? parseInt(saved) : 90;
      Alpine.store('heatmap', { range: this.range });
    },
    setRange(days) {
      this.range = days;
      localStorage.setItem('hallmonitor_timeRange_metric', days.toString());
      Alpine.store('heatmap', { range: days });
      window.dispatchEvent(new CustomEvent('rangeChanged'));
    }
  }
}
</script>
```

**Steps:**
1. Add Alpine `x-data` for heatmap state
2. Replace `onclick="setTimeRange()"` with `@click="setRange()"`
3. Add `:class` binding for active state
4. Trigger htmx refresh when range changes
5. Update backend to accept `range` query param
6. Test time range switching works
7. Remove old `setTimeRange()` and `updateTimeRangeButtons()`
8. Commit: "refactor: migrate heatmap time range to Alpine.js"

**Files Modified:**
- `internal/api/dashboard.html`
- `internal/api/partials.go` (heatmap handler to accept range param)

**Validation:**
- Click 7d/30d/90d buttons
- Active button highlighted
- Heatmap refreshes with new range
- Range persists on page reload

---

## 📁 Final File Structure

```
hallmonitor/
├── internal/
│   └── api/
│       ├── dashboard.html           # Main dashboard (htmx + Alpine)
│       ├── dashboard_ambient.html   # Ambient view (htmx + Alpine)
│       ├── handlers.go              # Existing handlers
│       ├── partials.go              # NEW: Partial template handlers
│       ├── server.go                # Updated routes
│       └── partials/                # NEW: Go template partials
│           ├── hero.html
│           ├── monitors.html
│           └── heatmap.html
└── go.mod
```

---

## 🧪 Testing Checklist (After Each Phase)

### **Functional Tests**
- [ ] Dashboard loads without errors
- [ ] Theme toggle works (dark/light)
- [ ] Monitor table renders correctly
- [ ] Row expansion works (click to expand/collapse)
- [ ] Search filter works
- [ ] Auto-refresh works every 30s
- [ ] Heatmap time range selector works
- [ ] Manual refresh button works
- [ ] View switching works (metric ↔ ambient)
- [ ] No JavaScript console errors

### **Performance Tests**
- [ ] Initial page load < 2s
- [ ] Auto-refresh doesn't cause page jump
- [ ] Smooth animations (theme, expand, etc.)
- [ ] No memory leaks (check DevTools)

### **Browser Compatibility**
- [ ] Chrome/Edge (latest)
- [ ] Firefox (latest)
- [ ] Safari (latest)
- [ ] Mobile Safari (iOS)
- [ ] Mobile Chrome (Android)

---

## 📊 Complexity Reduction Metrics

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| **Total JS Functions** | 29 | ~15 | -48% |
| **Lines of JS** | ~800 | ~400 | -50% |
| **Manual DOM Updates** | 10+ | 0 | -100% |
| **setInterval Calls** | 2 | 0 | -100% |
| **Event Listeners** | 15+ | ~5 | -67% |
| **State Variables** | 8 | 3 | -63% |

---

## 🎯 What We Keep vs What We Migrate

### **Keep in JavaScript**
- Utility functions (formatTimeAgo, parseDuration, etc.)
- Complex calculations (heatmap grid layout)
- One-time initialization
- Data transformations

### **Migrate to Alpine**
- UI state (theme, expanded rows, search query)
- User interactions (clicks, input changes)
- Conditional rendering
- CSS class toggling

### **Migrate to htmx**
- Data fetching (auto-refresh, manual refresh)
- Partial page updates
- Server-driven rendering

### **Migrate to Go Templates**
- HTML rendering (hero, table, heatmap)
- Data formatting (where appropriate)
- Partials for reusable sections

---

## 🚨 Rollback Plan

If anything goes wrong during migration:

1. **Git is your friend:**
   ```bash
   git status  # See what changed
   git diff    # Review changes
   git restore <file>  # Restore specific file
   git reset --hard HEAD  # Nuclear option: restore everything
   ```

2. **Each phase is a commit:**
   - Phase goes wrong? `git reset --hard HEAD~1`
   - Lost? Check git log: `git log --oneline`

3. **Feature flags (optional):**
   - Add `?legacy=1` URL param to use old JavaScript
   - Easy A/B testing

---

## ✅ Success Criteria

Migration is successful when:

1. **All features work** - Nothing breaks
2. **Less JavaScript** - 50% reduction in JS code
3. **Better UX** - Smoother interactions, no page flash
4. **Easier to maintain** - Clear separation of concerns
5. **Still beautiful** - Zero design changes

---

## 🎉 Expected Benefits

### **Developer Experience**
- ✅ Less manual DOM manipulation
- ✅ Declarative UI updates
- ✅ Server-driven rendering reduces client complexity
- ✅ Alpine handles reactivity automatically
- ✅ htmx eliminates manual fetch() calls

### **User Experience**
- ✅ Smoother interactions (Alpine animations)
- ✅ Faster perceived performance (partial updates)
- ✅ No full page reloads
- ✅ Progressive enhancement (works without JS)

### **Code Quality**
- ✅ Separation of concerns (server vs client)
- ✅ Less state management complexity
- ✅ Easier to test (Go templates testable)
- ✅ Smaller bundle size

---

## 📝 Notes

- **No breaking changes** - Each phase is backward compatible
- **Progressive enhancement** - Dashboard works with JS disabled (mostly)
- **Keep it simple** - No build step, no bundlers, no frameworks
- **Standard library** - Use Go's `html/template` for partials
- **CDN delivery** - htmx + Alpine from CDN (can self-host later)

---

**Created:** 2025-01-15
**Author:** Claude
**Status:** Ready to execute
**Estimated Time:** 2-4 hours (all phases)
