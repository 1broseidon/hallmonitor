// Dashboard JS - v2.0 with null checks
console.log('Dashboard JS loaded - v2.0 with null checks');

// Alpine.js theme manager
function themeManager() {
    return {
        init() {
            const savedTheme = localStorage.getItem('hallmonitor_theme') || 'dark';
            document.documentElement.dataset.theme = savedTheme;
        },
        toggle() {
            const html = document.documentElement;
            const newTheme = html.dataset.theme === 'dark' ? 'light' : 'dark';
            html.dataset.theme = newTheme;
            localStorage.setItem('hallmonitor_theme', newTheme);
        }
    };
}

// Alpine.js heatmap range manager
function heatmapRangeManager() {
    return {
        range: 90,
        init() {
            const saved = localStorage.getItem('hallmonitor_timeRange_metric');
            this.range = saved ? parseInt(saved) : 90;
            currentTimeRange = this.range;
        },
        setRange(days) {
            this.range = days;
            currentTimeRange = days;
            localStorage.setItem('hallmonitor_timeRange_metric', days.toString());
            generateHeatmap(uptimeHistory);
        }
    };
}

const API_ENDPOINT = '/api/v1';
const SLA_THRESHOLD = 99.9;
let monitorsData = null;
let uptimeHistory = null;
let currentTimeRange = 90; // Default to 90 days for metric view

// Resize handler for heatmap recalculation
let resizeTimeout;
function handleResize() {
    clearTimeout(resizeTimeout);
    resizeTimeout = setTimeout(() => {
        if (uptimeHistory !== null) {
            generateHeatmap(uptimeHistory);
        }
    }, 250); // Debounce resize events
}

// Initialize on load
document.addEventListener('DOMContentLoaded', async () => {
    await loadData();
    await loadUptimeHistory();
    // Note: Auto-refresh now handled by htmx (every 30s)
    // Note: Time range now managed by Alpine heatmapRangeManager()

    // Add resize listener for responsive heatmap
    window.addEventListener('resize', handleResize);
});

async function loadData() {
    try {
        const response = await fetch(`${API_ENDPOINT}/monitors`);
        const data = await response.json();
        monitorsData = data;
        updateUI(data);
    } catch (error) {
        console.error('Failed to load data:', error);
    }
}

async function loadUptimeHistory() {
    try {
        // Fetch historical data from new history API
        const monitors = monitorsData?.monitors || [];
        if (monitors.length === 0) {
            generateHeatmap(null);
            return;
        }

        // Get history for the first monitor as a sample (can be extended to all monitors)
        const monitorName = monitors[0]?.name;
        if (!monitorName) {
            generateHeatmap(null);
            return;
        }

        // Fetch last 30 days of history
        const end = new Date();
        const start = new Date();
        start.setDate(start.getDate() - 30);

        const response = await fetch(
            `${API_ENDPOINT}/monitors/${encodeURIComponent(monitorName)}/history?` +
            `start=${start.toISOString()}&end=${end.toISOString()}&limit=10000`
        );

        if (!response.ok) {
            throw new Error(`HTTP error ${response.status}`);
        }

        const data = await response.json();
        uptimeHistory = parseHistoricalResults(data.results || []);
        generateHeatmap(uptimeHistory);
    } catch (error) {
        console.error('Failed to load uptime history:', error);
        // Fall back to simulated data for demo
        generateHeatmap(null);
    }
}

function parseHistoricalResults(results) {
    // Convert historical results into daily uptime data
    const uptimeData = {};

    for (const result of results) {
        const date = result.timestamp.split('T')[0];
        if (!uptimeData[date]) {
            uptimeData[date] = { up: 0, down: 0, total: 0 };
        }

        uptimeData[date].total++;
        if (result.status === 'up') {
            uptimeData[date].up++;
        } else {
            uptimeData[date].down++;
        }
    }

    // Convert to array of uptime values (0-1 range)
    const dailyUptimes = {};
    for (const [date, stats] of Object.entries(uptimeData)) {
        dailyUptimes[date] = stats.total > 0 ? stats.up / stats.total : 1.0;
    }

    return dailyUptimes;
}

async function loadUptimeStats() {
    try {
        // Load uptime statistics for different periods
        const monitors = monitorsData?.monitors || [];
        if (monitors.length === 0) return;

        const monitorName = monitors[0]?.name;
        if (!monitorName) return;

        // Load 24h, 7d, and 30d uptime stats in parallel
        const [uptime24h, uptime7d, uptime30d] = await Promise.all([
            fetch(`${API_ENDPOINT}/monitors/${encodeURIComponent(monitorName)}/uptime?period=24h`)
                .then(r => r.ok ? r.json() : null),
            fetch(`${API_ENDPOINT}/monitors/${encodeURIComponent(monitorName)}/uptime?period=168h`)
                .then(r => r.ok ? r.json() : null),
            fetch(`${API_ENDPOINT}/monitors/${encodeURIComponent(monitorName)}/uptime?period=720h`)
                .then(r => r.ok ? r.json() : null)
        ]);

        // Store in global variable for use in UI
        window.uptimeStats = {
            '24h': uptime24h,
            '7d': uptime7d,
            '30d': uptime30d
        };

        console.log('Uptime stats loaded:', window.uptimeStats);
    } catch (error) {
        console.error('Failed to load uptime stats:', error);
    }
}

// htmx refresh handler - called when htmx auto-refresh completes
function htmxRefreshHandler(event) {
    try {
        const data = JSON.parse(event.detail.xhr.responseText);
        monitorsData = data;
        updateUI(data);
    } catch (error) {
        console.error('htmx refresh failed:', error);
    }
}

async function refreshData() {
    await loadData();
    await loadUptimeHistory();
    await loadUptimeStats();
}

function exportToGrafana() {
    window.open('/api/v1/grafana/dashboard', '_blank');
}

function parseDuration(durationStr) {
    if (!durationStr) return 0;
    const match = durationStr.match(/([\d.]+)(ms|s|µs)/);
    if (!match) return 0;
    const value = parseFloat(match[1]);
    const unit = match[2];
    if (unit === 's') return value * 1000;
    if (unit === 'µs') return value / 1000;
    return value;
}

function updateUI(data) {
    const monitors = data.monitors || [];
    const total = monitors.length;
    const down = monitors.filter(m => m.status !== 'up').length;
    const uptime = total > 0 ? ((total - down) / total * 100) : 100;

    // Calculate response times
    const responseTimes = monitors
        .map(m => parseDuration(m.duration))
        .filter(t => t > 0);

    const avgResponse = responseTimes.length > 0
        ? (responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length)
        : 0;

    // Calculate P95
    const sortedTimes = [...responseTimes].sort((a, b) => a - b);
    const p95Index = Math.floor(sortedTimes.length * 0.95);
    const p95 = sortedTimes[p95Index] || 0;

    // Calculate total checks and error rate
    const totalChecks = monitors.reduce((sum, m) => sum + (m.total_checks || 0), 0);
    const totalErrors = monitors.reduce((sum, m) => sum + (m.error_count || 0), 0);
    const errorRate = totalChecks > 0 ? (totalErrors / totalChecks * 100) : 0;

	// Update hero metric
	const heroUptimeEl = document.getElementById('hero-uptime');
	if (heroUptimeEl) {
		heroUptimeEl.textContent = uptime.toFixed(2);
	}
	const heroSubEl = document.getElementById('hero-sublabel');
	if (heroSubEl) {
		heroSubEl.textContent = `${total - down} of ${total} monitors healthy`;
	}
	const heroValue = document.getElementById('hero-ring');
	const heroCaption = document.querySelector('.hero-caption');
	const heroNarrative = document.getElementById('hero-narrative');
	const meetsSLA = uptime >= SLA_THRESHOLD;
	const glowColor = meetsSLA ? '72,199,142' : '241,70,104';
	if (heroValue) {
		heroValue.style.setProperty('--hero-glow', glowColor);
	}
	if (heroCaption) {
		heroCaption.textContent = meetsSLA ? 'uptime' : 'below target';
	}
	if (heroNarrative) {
		heroNarrative.textContent = meetsSLA ? 'All services are within expected ranges.' : `${down} monitor${down > 1 ? 's' : ''} degraded · see Focus list.`;
	}

	// Update compact metrics (with null checks)
	const p95ValueEl = document.getElementById('p95-value');
	if (p95ValueEl) p95ValueEl.textContent = p95.toFixed(1) + 'ms';

	const monitorsValueEl = document.getElementById('monitors-value');
	if (monitorsValueEl) monitorsValueEl.textContent = total;

	const checksValueEl = document.getElementById('checks-value');
	if (checksValueEl) checksValueEl.textContent = formatNumber(totalChecks);

	const errorRateValueEl = document.getElementById('error-rate-value');
	if (errorRateValueEl) {
		errorRateValueEl.textContent = errorRate.toFixed(2) + '%';
		// Update color
		errorRateValueEl.style.color = errorRate === 0 ? '#48c78e' : errorRate < 1 ? '#ffdd57' : '#f14668';
	}

	const heroMonitors = document.getElementById('hero-pill-monitors');
	const heroIncidents = document.getElementById('hero-pill-incidents');
	if (heroMonitors) heroMonitors.textContent = `${total - down}/${total}`;
	if (heroIncidents) heroIncidents.textContent = down;

	const heroLastAlert = document.getElementById('hero-last-alert');
	if (heroLastAlert) {
		if (down === 0) {
			heroLastAlert.textContent = 'All systems steady';
		} else {
			const troubled = monitors.find(m => m.status !== 'up');
			heroLastAlert.textContent = troubled ? `${troubled.name} · ${troubled.group || troubled.type}` : `${down} monitors degraded`;
		}
	}

    // Update monitor list
    updateMonitorList(monitors);
}

let filteredMonitors = [];
let displayedCount = 50;
const batchSize = 20;
let isLoading = false;

function handleSearch(query) {
    const searchQuery = query.toLowerCase();
    const allMonitors = monitorsData?.monitors || [];

    if (searchQuery === '') {
        filteredMonitors = [...allMonitors];
    } else {
        filteredMonitors = allMonitors.filter(monitor =>
            monitor.name.toLowerCase().includes(searchQuery) ||
            monitor.type.toLowerCase().includes(searchQuery) ||
            monitor.group.toLowerCase().includes(searchQuery) ||
            (monitor.target && monitor.target.toLowerCase().includes(searchQuery)) ||
            (monitor.url && monitor.url.toLowerCase().includes(searchQuery)) ||
            (monitor.hostname && monitor.hostname.toLowerCase().includes(searchQuery)) ||
            (monitor.ip_address && monitor.ip_address.toLowerCase().includes(searchQuery)) ||
            (monitor.query && monitor.query.toLowerCase().includes(searchQuery)) ||
            (monitor.labels && Object.values(monitor.labels).some(v => v.toLowerCase().includes(searchQuery)))
        );
    }

    displayedCount = Math.min(50, filteredMonitors.length);
    renderTable();
}

function formatTimeAgo(timestamp) {
    if (!timestamp) return 'Never';
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now - date;
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);

    if (diffSec < 60) return `${diffSec}s ago`;
    if (diffMin < 60) return `${diffMin}m ago`;
    if (diffHour < 24) return `${diffHour}h ago`;
    return `${Math.floor(diffHour / 24)}d ago`;
}

function formatDate(value) {
    if (!value) return '—';
    const date = new Date(value);
    if (isNaN(date.getTime())) {
        return value;
    }
    return date.toLocaleDateString();
}

function updateMonitorList(monitors) {
    filteredMonitors = [...monitors];
    displayedCount = Math.min(50, filteredMonitors.length);
    renderTable();
}

function renderTable() {
    const tableBody = document.getElementById('tableBody');
    const visibleMonitors = filteredMonitors.slice(0, displayedCount);

    if (visibleMonitors.length === 0) {
        tableBody.innerHTML = `
            <tr>
                <td colspan="6">
                    <div class="empty-state">
                        <i class="fas fa-search"></i>
                        <p>No monitors found</p>
                    </div>
                </td>
            </tr>
        `;
        return;
    }

    // Sort: down monitors first, then by name
    const sortedMonitors = [...visibleMonitors].sort((a, b) => {
        const aUp = a.status === 'up' ? 1 : 0;
        const bUp = b.status === 'up' ? 1 : 0;
        if (aUp !== bUp) return aUp - bUp;
        return a.name.localeCompare(b.name);
    });

    tableBody.innerHTML = sortedMonitors.map(monitor => {
        const isUp = monitor.status === 'up';
        const responseTime = parseDuration(monitor.duration);

        // Calculate uptime (simplified - shows 100% if up, 0% if down)
        // Real uptime would require historical data
        const uptime = isUp ? 100.000 : 0.000;
        const uptimeClass = uptime >= 99.9 ? 'success' : uptime >= 99 ? 'warning' : 'danger';
        const responseClass = responseTime < 100 ? 'success' : responseTime < 500 ? 'warning' : 'danger';

        // Get target/URL for display
        const target = monitor.url || monitor.target || 'N/A';

        // Use extracted hostname and IP if available, otherwise fall back to parsing
        const hostname = monitor.hostname || null;
        const ip = monitor.ip_address || (monitor.target && /^\d+\.\d+\.\d+\.\d+/.test(monitor.target) ? monitor.target : null);

        // Get best display value for target column (prefer hostname, then IP, then URL/target)
        let targetDisplay = 'N/A';
        if (hostname) {
            targetDisplay = hostname;
        } else if (ip) {
            targetDisplay = ip;
        } else if (monitor.url) {
            // Extract hostname from URL for cleaner display
            try {
                const urlObj = new URL(monitor.url);
                targetDisplay = urlObj.hostname;
            } catch {
                targetDisplay = monitor.url;
            }
        } else if (monitor.target) {
            // For TCP monitors, show host:port format
            if (monitor.target.includes(':')) {
                targetDisplay = monitor.target;
            } else {
                targetDisplay = monitor.target;
            }
        } else if (monitor.query) {
            // For DNS monitors, show query
            targetDisplay = monitor.query;
        }

        // Get type-specific data
        const httpResult = monitor.http_result;
        const pingResult = monitor.ping_result;
        const tcpResult = monitor.tcp_result;
        const dnsResult = monitor.dns_result;

        const monitorId = escapeHtml(monitor.name);
        return `
            <tr class="main-row"
                :class="{ 'expanded': expandedRow === '${monitorId}' }"
                @click="expandedRow = expandedRow === '${monitorId}' ? null : '${monitorId}'">
                <td>
                    <div class="monitor-name-cell">
                        <span class="status-dot ${monitor.status}"></span>
                        <div class="monitor-primary">
                            <span class="monitor-name">${monitor.name}</span>
                            <span class="monitor-type">${monitor.type}</span>
                        </div>
                    </div>
                </td>
                <td>
                    <span style="font-size: 0.875rem; opacity: 0.8; font-family: 'JetBrains Mono', monospace;">${targetDisplay}</span>
                </td>
                <td><span class="metric-value ${uptimeClass}">${uptime.toFixed(3)}%</span></td>
                <td><span class="metric-value ${responseClass}">${responseTime > 0 ? responseTime.toFixed(1) : '--'}ms</span></td>
                <td><span style="opacity: 0.5; font-size: 0.9375rem;">${formatTimeAgo(monitor.last_check)}</span></td>
                <td class="expand-cell">
                    <div class="expand-icon">
                        <i class="fas fa-chevron-down"></i>
                    </div>
                </td>
            </tr>
            <tr class="detail-row"
                x-show="expandedRow === '${monitorId}'"
                x-collapse>
                <td colspan="6">
                    <div class="detail-content">
                        ${renderDetailBlocks(monitor)}
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

function renderDetailBlocks(monitor) {
    const blocks = [];
    const addBlock = (label, value, options = {}) => {
        if (value === undefined || value === null || value === '') {
            return;
        }
        const full = options.full ? ' style="grid-column: 1 / -1;"' : '';
        const raw = options.raw || false;
        const displayValue = raw ? value : escapeHtml(String(value));
        blocks.push(`
            <div class="detail-group"${full}>
                <span class="detail-label">${label}</span>
                <span class="detail-value">${displayValue}</span>
            </div>
        `);
    };

    addBlock('Last check', formatTimeAgo(monitor.last_check));
    addBlock('Monitor type', monitor.type?.toUpperCase());
    addBlock('Current status', monitor.status === 'up' ? '<span style="color:#48c78e">Up</span>' : '<span style="color:#f14668">Down</span>', { raw: true });

    const target = monitor.url || monitor.target || monitor.query;
    if (target) {
        addBlock('Target', target, { full: true });
    }

    if (monitor.error) {
        addBlock('Current error', `<span style="color:#f14668;">${escapeHtml(monitor.error)}</span>`, { raw: true, full: true });
    }

    const httpResult = monitor.http_result || {};
    if (httpResult.status_code) {
        addBlock('HTTP status', `${httpResult.status_code}`);
    }
    if (httpResult.response_size) {
        addBlock('Response size', `${httpResult.response_size} bytes`);
    }
    if (httpResult.ssl_cert_expiry) {
        addBlock('SSL expires', formatDate(httpResult.ssl_cert_expiry));
    }

    const pingResult = monitor.ping_result || {};
    if (pingResult.packet_loss !== undefined) {
        addBlock('Packet loss', `${pingResult.packet_loss.toFixed(1)}%`);
    }
    if (pingResult.avg_rtt) {
        addBlock('Average RTT', `${pingResult.avg_rtt}`);
    }

    const tcpResult = monitor.tcp_result || {};
    if (tcpResult.port) {
        addBlock('TCP port', tcpResult.port);
    }
    if (tcpResult.response_time) {
        addBlock('Connect time', `${tcpResult.response_time}`);
    }

    const dnsResult = monitor.dns_result || {};
    if (dnsResult.query_type) {
        addBlock('DNS query type', dnsResult.query_type);
    }
    if (Array.isArray(dnsResult.answers) && dnsResult.answers.length > 0) {
        const answers = dnsResult.answers.map(ans => `<span class="tag">${escapeHtml(ans)}</span>`).join('');
        addBlock('DNS answers', `<div style="margin-top:0.25rem; display:flex; flex-wrap:wrap; gap:0.25rem;">${answers}</div>`, { raw: true, full: true });
    }

    if (monitor.labels && Object.keys(monitor.labels).length) {
        const tags = Object.entries(monitor.labels).map(([key, value]) => `<span class="tag">${escapeHtml(key)}:${escapeHtml(value)}</span>`).join('');
        addBlock('Labels', `<div style="margin-top:0.25rem; display:flex; flex-wrap:wrap; gap:0.25rem;">${tags}</div>`, { raw: true, full: true });
    }

    if (monitor.metadata && typeof monitor.metadata === 'object') {
        const entries = Object.entries(monitor.metadata).map(([key, value]) => `<span class="tag">${escapeHtml(key)}:${escapeHtml(String(value))}</span>`).join('');
        if (entries) {
            addBlock('Metadata', `<div style="margin-top:0.25rem; display:flex; flex-wrap:wrap; gap:0.25rem;">${entries}</div>`, { raw: true, full: true });
        }
    }

    if (!blocks.length) {
        addBlock('Details', 'No additional data');
    }

    return blocks.join('');
}

function escapeHtml(value) {
    if (typeof value !== 'string') return value;
    return value
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function calculateUptimeLevel(uptimeValue) {
    // Convert 0-1 range to 1-5 levels with better thresholds
    if (uptimeValue < 0.5) return 1;      // <50% - Critical (red)
    if (uptimeValue < 0.8) return 2;      // 50-80% - Poor (orange/yellow)
    if (uptimeValue < 0.9) return 3;      // 80-90% - Warning (yellow)
    if (uptimeValue < 0.98) return 4;     // 90-98% - Good (light green)
    return 5;                              // 98-100% - Excellent (green)
}

function calculateOptimalGridLayout(days, containerWidth, containerHeight) {
    const gap = 3;

    // Try different column counts to find the best fit
    // We want to maximize tile size while keeping them square
    let bestColumns = 7;
    let bestTileSize = 0;

    // Try column counts from 5 to 30
    for (let cols = 5; cols <= Math.min(30, days); cols++) {
        const rows = Math.ceil(days / cols);

        // Calculate tile size based on width
        const availableWidth = containerWidth - (gap * (cols - 1));
        const tileWidth = availableWidth / cols;

        // Calculate tile size based on height
        const availableHeight = containerHeight - (gap * (rows - 1));
        const tileHeight = availableHeight / rows;

        // Use the smaller dimension to keep square
        const tileSize = Math.min(tileWidth, tileHeight);

        // Prefer layouts that maximize tile size
        if (tileSize > bestTileSize && tileSize >= 8) { // Minimum 8px tiles
            bestTileSize = tileSize;
            bestColumns = cols;
        }
    }

    const rows = Math.ceil(days / bestColumns);

    return { columns: bestColumns, rows };
}

function generateHeatmap(historyData) {
    const grid = document.getElementById('heatmap-grid');
    if (!grid) return;

    // Use requestAnimationFrame to ensure container is sized
    requestAnimationFrame(() => {
        grid.innerHTML = '';

        // Use currentTimeRange (7, 30, or 90 days)
        const days = currentTimeRange;
        const today = new Date();
        today.setHours(23, 59, 59, 999); // Set to end of today for proper comparison
        const todayStr = today.toISOString().split('T')[0];

        // Calculate optimal grid layout based on container dimensions
        // Account for padding (2rem = 32px on each side) and gap
        const card = grid.closest('.heatmap-card');
        const cardWidth = card ? card.clientWidth : 800;
        const cardHeight = grid.clientHeight || 300; // Use grid height if available
        const padding = 64; // 2rem * 2 (left + right)
        const availableWidth = Math.max(200, cardWidth - padding);
        const availableHeight = Math.max(150, cardHeight - 20); // Account for header
        const { columns, rows } = calculateOptimalGridLayout(days, availableWidth, availableHeight);

        // Set grid layout to fill container
        grid.style.gridTemplateColumns = `repeat(${columns}, 1fr)`;
        grid.style.gridAutoRows = 'auto'; // Let rows size naturally based on aspect-ratio

        _generateHeatmapCells(grid, days, today, todayStr, historyData);
    });
}

function _generateHeatmapCells(grid, days, today, todayStr, historyData) {

    for (let i = 0; i < days; i++) {
        const cell = document.createElement('div');

        const date = new Date(today);
        date.setDate(date.getDate() - (days - 1 - i));
        date.setHours(0, 0, 0, 0); // Set to start of day for comparison
        const dateStr = date.toISOString().split('T')[0];

        // Determine state: past-no-data, future, or has-data
        const isFuture = date > today;
        const hasHistoricalData = historyData && historyData[dateStr] !== undefined;
        const isToday = dateStr === todayStr;

        let tooltipText;

        if (isFuture) {
            // Future date - not yet occurred
            cell.className = 'heatmap-cell level-future';
            tooltipText = `${date.toLocaleDateString()}: Not yet occurred`;
        } else if (hasHistoricalData) {
            // Has actual historical data from BadgerDB
            const uptimeValue = historyData[dateStr]; // 0-1 range
            const level = calculateUptimeLevel(uptimeValue);
            const uptimePercent = (uptimeValue * 100).toFixed(1);
            cell.className = `heatmap-cell level-${level}`;
            tooltipText = `${date.toLocaleDateString()}: ${uptimePercent}% uptime`;
        } else if (isToday && monitorsData) {
            // Current day - use live monitor status
            const monitors = monitorsData.monitors || [];
            const total = monitors.length;
            const up = monitors.filter(m => m.status === 'up').length;
            const uptimeValue = total > 0 ? (up / total) : 1.0;
            const level = calculateUptimeLevel(uptimeValue);
            const uptimePercent = (uptimeValue * 100).toFixed(1);
            cell.className = `heatmap-cell level-${level}`;
            tooltipText = `${date.toLocaleDateString()}: ${uptimePercent}% uptime (current)`;
        } else {
            // Past date with no data - Hall Monitor wasn't running
            cell.className = 'heatmap-cell level-past-nodata';
            tooltipText = `${date.toLocaleDateString()}: No data (Hall Monitor not running)`;
        }

        cell.title = tooltipText;
        grid.appendChild(cell);
    }
}
