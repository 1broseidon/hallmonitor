# Discord Alerts - 5-Minute Quick Start

## Step 1: Get Discord Webhook (2 minutes)

1. Open Discord
2. Go to your server → **Server Settings** → **Integrations** → **Webhooks**
3. Click **"New Webhook"** or **"Create Webhook"**
4. Name it: `Hall Monitor Alerts`
5. Select the channel for alerts
6. Click **"Copy Webhook URL"**

URL format: `https://discord.com/api/webhooks/123456.../AbCdEf...`

## Step 2: Configure Webhook (30 seconds)

```bash
# Edit .env file
nano .env

# Add this line (paste your webhook URL):
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_WEBHOOK_HERE
```

Save and exit (Ctrl+X, Y, Enter)

## Step 3: Restart Adapter (30 seconds)

```bash
docker restart discord-adapter

# Verify it's working:
docker logs discord-adapter --tail 5
```

You should see: `✅ Configured` in the logs

## Step 4: Test It! (2 minutes)

### Quick Test - Send Test Alert

```bash
curl -X POST http://localhost:19093/api/v1/alerts \
  -H 'Content-Type: application/json' -d '
[
  {
    "labels": {
      "alertname": "TestAlert",
      "severity": "warning",
      "monitor": "test-system",
      "component": "testing"
    },
    "annotations": {
      "summary": "🎉 Discord integration is working!",
      "description": "You successfully configured Hall Monitor alerts!"
    }
  }
]'
```

**Check Discord in 30 seconds** - you should see a warning alert! 🎉

## That's It!

Your alerting system is now live. Alerts will automatically appear in Discord when:
- Monitors go down
- Response times are high
- SSL certificates expiring
- Packet loss detected
- High error rates
- And 13 other conditions!

## View Active Alerts

- **Prometheus**: http://localhost:19090/alerts
- **Alertmanager**: http://localhost:19093
- **Grafana Dashboard**: http://localhost:3000

## Need Help?

Full documentation: `deploy/observability/ALERTING_SETUP.md`

## Common Issues

**"ERROR: DISCORD_WEBHOOK_URL not set"**
- Make sure you saved .env file
- Restart: `docker restart discord-adapter`

**"401 Unauthorized" in Discord adapter logs**
- Webhook URL is invalid or expired
- Create new webhook in Discord and update .env

**Alerts not showing in Discord**
- Verify webhook works: Send test message to webhook URL directly
- Check adapter logs: `docker logs discord-adapter`
- Ensure no typos in webhook URL

---

**That's all you need!** 🚀

