// Ambient Dashboard JS - zen mode
console.log('Ambient Dashboard JS loaded - zen mode');

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

const API_ENDPOINT = '/api/v1';
let monitorsData = null;

// htmx refresh handler
function htmxRefreshHandler(event) {
    try {
        const data = JSON.parse(event.detail.xhr.responseText);
        monitorsData = data;
        updateZenUI(data);
    } catch (error) {
        console.error('htmx refresh failed:', error);
    }
}

async function loadData() {
    try {
        const response = await fetch(`${API_ENDPOINT}/monitors`);
        const data = await response.json();
        monitorsData = data;
        updateZenUI(data);
    } catch (error) {
        console.error('Failed to load data:', error);
    }
}

function updateZenUI(data) {
    const monitors = data.monitors || [];
    const total = monitors.length;
    const down = monitors.filter(m => m.status !== 'up').length;
    const uptime = total > 0 ? ((total - down) / total * 100) : 100;

    // Update giant uptime
    const zenUptimeEl = document.getElementById('zen-uptime');
    if (zenUptimeEl) {
        zenUptimeEl.textContent = uptime.toFixed(2) + '%';

        // Update gradient and glow color based on health
        if (uptime >= 99.9) {
            zenUptimeEl.style.background = 'linear-gradient(135deg, #48c78e 0%, #667eea 100%)';
            zenUptimeEl.style.setProperty('--zen-glow', '72,199,142'); // Green glow
        } else if (uptime >= 95) {
            zenUptimeEl.style.background = 'linear-gradient(135deg, #ffdd57 0%, #667eea 100%)';
            zenUptimeEl.style.setProperty('--zen-glow', '255,221,87'); // Yellow glow
        } else {
            zenUptimeEl.style.background = 'linear-gradient(135deg, #f14668 0%, #667eea 100%)';
            zenUptimeEl.style.setProperty('--zen-glow', '241,70,104'); // Red glow
        }
        zenUptimeEl.style.webkitBackgroundClip = 'text';
        zenUptimeEl.style.webkitTextFillColor = 'transparent';
        zenUptimeEl.style.backgroundClip = 'text';
    }

    // Update monitors count
    const zenMonitorsEl = document.getElementById('zen-monitors');
    if (zenMonitorsEl) {
        zenMonitorsEl.textContent = total;
    }

    // Update incidents
    const zenIncidentsEl = document.getElementById('zen-incidents');
    if (zenIncidentsEl) {
        zenIncidentsEl.textContent = down;
        zenIncidentsEl.style.color = down === 0 ? '#48c78e' : '#f14668';
    }

    // Update message
    const zenMessageEl = document.getElementById('zen-message');
    if (zenMessageEl) {
        if (down === 0) {
            zenMessageEl.textContent = 'All systems operational';
        } else if (down === 1) {
            const troubled = monitors.find(m => m.status !== 'up');
            zenMessageEl.textContent = troubled ? `${troubled.name} is experiencing issues` : '1 monitor degraded';
        } else {
            zenMessageEl.textContent = `${down} monitors require attention`;
        }
    }
}

// Initialize on load
document.addEventListener('DOMContentLoaded', async () => {
    await loadData();
});
