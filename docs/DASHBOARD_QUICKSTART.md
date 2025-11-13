# Dashboard Quick Start

## 🚀 Getting Started in 30 Seconds

1. **Start Hall Monitor:**
```bash
./hallmonitor --config config.yml
```

2. **Open your browser:**
```
http://localhost:7878
```

That's it! You now have a beautiful monitoring dashboard.

## 📸 What You'll See

### Hero Stats
Four big numbers that tell you everything:
- ✅ **99.999%** - Your uptime (green = good, red = trouble)
- ⚠️ **0** - Active incidents (0 = sleeping well tonight)
- ⚡ **25ms** - Average response time
- 📊 **6** - Total monitors running

### Visual Health Matrix
Instantly see which services are up (green) or down (red) organized by type.

### Currently Down Section
Red alert box appears only when something is broken. Otherwise shows "All Systems Operational ✓"

### Performance Charts
Beautiful graphs showing response times trending over time.

## 🎨 Features

- **Auto-refresh** every 30 seconds
- **Mobile-friendly** design
- **Dark mode** support (if your browser has it)
- **Export to Grafana** for power users
- **Zero configuration** required

## 🔧 Configuration

### Enable/Disable Dashboard

In `config.yml`:
```yaml
server:
  enableDashboard: true  # Set to false to disable
```

Or via environment variable:
```bash
export SERVER_ENABLEDASHBOARD=false
./hallmonitor
```

### Change Port

```yaml
server:
  port: "8080"  # Use any port you want
```

Then access at: `http://localhost:8080`

## 🎓 For Advanced Users

### Export to Grafana

1. Click **"Export to Grafana"** button in top-right
2. Import the JSON into your Grafana instance
3. Get advanced features:
   - Custom time ranges
   - Advanced alerting
   - Historical analysis
   - Panel customization

The Grafana dashboard we built includes:
- ✅ Interactive drill-down (click type → filter to that type)
- ✅ Click monitor name → see just that monitor
- ✅ Beautiful enterprise-grade visuals
- ✅ All the charts and metrics you need

### Using Your Own Dashboard

Prefer to build your own? No problem:

1. Set `enableDashboard: false` in config
2. Query the API directly:
   - `GET /api/v1/monitors` - All monitor statuses
   - `GET /metrics` - Prometheus metrics
   - `GET /health` - Health check

3. Build whatever you want with the data!

## 🐛 Troubleshooting

**Dashboard shows "No monitors configured"**
- Add monitors to your `config.yml` file
- Restart Hall Monitor

**Data not updating**
- Check `/health` endpoint is returning OK
- Verify monitors are actually running
- Check browser console for errors

**Want the old API-only behavior**
```yaml
server:
  enableDashboard: false
```

## 💡 Pro Tips

1. **Bookmark it**: Add to your browser favorites for quick access
2. **Multiple instances**: Run multiple Hall Monitors on different ports
3. **Reverse proxy**: Put it behind nginx/traefik with SSL
4. **TV Dashboard**: Open on a TV/monitor for always-on visibility

## 🎯 Next Steps

- Read [DASHBOARD.md](./DASHBOARD.md) for full documentation
- Check out [GRAFANA_DASHBOARD_PLAN.md](../world-class-grafana-dashboard.plan.md) to see how we designed the Grafana version
- Browse the [deploy/](../deploy/) folder for Grafana/Prometheus/Loki setup

---

**Questions?** Open an issue on GitHub!

**Love it?** Give us a ⭐ star!

