#!/usr/bin/env python3
"""
Discord Webhook Adapter for Alertmanager
Translates Alertmanager webhook format to Discord webhook format
"""

import json
import os
import sys
from http.server import HTTPServer, BaseHTTPRequestHandler
from datetime import datetime
import requests

# Discord webhook URL from environment
DISCORD_WEBHOOK_URL = os.environ.get('DISCORD_WEBHOOK_URL', '')

# Color codes for Discord embeds
COLORS = {
    'critical': 0xFF0000,  # Red
    'warning': 0xFFA500,   # Orange
    'info': 0x00FF00,      # Green
    'resolved': 0x00FF00   # Green
}

def format_alert_for_discord(alert):
    """Convert Alertmanager alert to Discord embed format"""
    status = alert.get('status', 'firing')
    labels = alert.get('labels', {})
    annotations = alert.get('annotations', {})
    
    severity = labels.get('severity', 'info')
    monitor = labels.get('monitor', 'unknown')
    alert_name = labels.get('alertname', 'Alert')
    component = labels.get('component', 'system')
    
    # Determine color
    if status == 'resolved':
        color = COLORS['resolved']
        title_prefix = "✅ RESOLVED"
    elif severity == 'critical':
        color = COLORS['critical']
        title_prefix = "🔥 CRITICAL"
    elif severity == 'warning':
        color = COLORS['warning']
        title_prefix = "⚠️ WARNING"
    else:
        color = COLORS['info']
        title_prefix = "ℹ️ INFO"
    
    # Build embed
    embed = {
        "title": f"{title_prefix}: {alert_name}",
        "description": annotations.get('summary', f"Alert {alert_name} is {status}"),
        "color": color,
        "fields": [],
        "timestamp": alert.get('startsAt', datetime.utcnow().isoformat())
    }
    
    # Add monitor info if available
    if monitor != 'unknown':
        embed['fields'].append({
            "name": "Monitor",
            "value": monitor,
            "inline": True
        })
    
    # Add component
    if component:
        embed['fields'].append({
            "name": "Component",
            "value": component,
            "inline": True
        })
    
    # Add severity
    embed['fields'].append({
        "name": "Severity",
        "value": severity.upper(),
        "inline": True
    })
    
    # Add description if available
    description = annotations.get('description')
    if description:
        embed['fields'].append({
            "name": "Details",
            "value": description[:1024],  # Discord limit
            "inline": False
        })
    
    # Add dashboard link if available
    dashboard = annotations.get('dashboard')
    if dashboard:
        embed['fields'].append({
            "name": "Dashboard",
            "value": f"[View Dashboard]({dashboard})",
            "inline": False
        })
    
    # Add other labels as a field
    other_labels = {k: v for k, v in labels.items() 
                   if k not in ['alertname', 'severity', 'monitor', 'component']}
    if other_labels:
        labels_str = ', '.join([f"{k}={v}" for k, v in other_labels.items()])
        embed['fields'].append({
            "name": "Labels",
            "value": labels_str[:1024],
            "inline": False
        })
    
    return embed

def send_to_discord(webhook_url, alerts):
    """Send alerts to Discord webhook"""
    if not webhook_url:
        print("ERROR: DISCORD_WEBHOOK_URL not configured", file=sys.stderr)
        return False
    
    # Group alerts by status
    firing = [a for a in alerts if a.get('status') == 'firing']
    resolved = [a for a in alerts if a.get('status') == 'resolved']
    
    embeds = []
    
    # Add firing alerts
    for alert in firing[:10]:  # Discord limit: 10 embeds
        embeds.append(format_alert_for_discord(alert))
    
    # Add resolved alerts
    for alert in resolved[:10 - len(embeds)]:
        embeds.append(format_alert_for_discord(alert))
    
    if not embeds:
        return True
    
    # Build Discord webhook payload
    payload = {
        "username": "Hall Monitor Alerts",
        "avatar_url": "https://raw.githubusercontent.com/prometheus/prometheus/main/documentation/images/prometheus-logo.svg",
        "embeds": embeds
    }
    
    # Add summary content
    firing_count = len(firing)
    resolved_count = len(resolved)
    summary_parts = []
    if firing_count:
        summary_parts.append(f"🔥 {firing_count} alert{'s' if firing_count != 1 else ''} firing")
    if resolved_count:
        summary_parts.append(f"✅ {resolved_count} resolved")
    
    if summary_parts:
        payload["content"] = " | ".join(summary_parts)
    
    try:
        response = requests.post(
            webhook_url,
            json=payload,
            headers={'Content-Type': 'application/json'},
            timeout=10
        )
        response.raise_for_status()
        print(f"Successfully sent {len(embeds)} alerts to Discord")
        return True
    except requests.exceptions.RequestException as e:
        print(f"ERROR sending to Discord: {e}", file=sys.stderr)
        return False

class WebhookHandler(BaseHTTPRequestHandler):
    """HTTP handler for Alertmanager webhooks"""
    
    def do_POST(self):
        """Handle POST requests from Alertmanager"""
        content_length = int(self.headers.get('Content-Length', 0))
        
        try:
            # Parse Alertmanager webhook payload
            body = self.rfile.read(content_length)
            data = json.loads(body)
            
            alerts = data.get('alerts', [])
            print(f"Received {len(alerts)} alerts from Alertmanager")
            
            # Send to Discord
            success = send_to_discord(DISCORD_WEBHOOK_URL, alerts)
            
            if success:
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps({"status": "ok"}).encode())
            else:
                self.send_response(500)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps({"status": "error"}).encode())
                
        except Exception as e:
            print(f"ERROR processing webhook: {e}", file=sys.stderr)
            self.send_response(500)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "error", "message": str(e)}).encode())
    
    def do_GET(self):
        """Handle GET requests (health check)"""
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps({
            "status": "healthy",
            "service": "Discord Webhook Adapter",
            "webhook_configured": bool(DISCORD_WEBHOOK_URL)
        }).encode())
    
    def log_message(self, format, *args):
        """Custom logging"""
        print(f"{self.address_string()} - {format % args}")

def main():
    """Start the webhook adapter server"""
    if not DISCORD_WEBHOOK_URL:
        print("=" * 70)
        print("ERROR: DISCORD_WEBHOOK_URL environment variable not set!")
        print("=" * 70)
        print("\nTo get a Discord webhook URL:")
        print("1. Open Discord and go to Server Settings")
        print("2. Navigate to Integrations > Webhooks")
        print("3. Click 'New Webhook'")
        print("4. Configure the webhook and copy the URL")
        print("5. Set the environment variable:")
        print("   export DISCORD_WEBHOOK_URL='https://discord.com/api/webhooks/...'")
        print("\nThen restart this adapter.")
        print("=" * 70)
        sys.exit(1)
    
    port = int(os.environ.get('WEBHOOK_PORT', 5001))
    server = HTTPServer(('0.0.0.0', port), WebhookHandler)
    
    print("=" * 70)
    print(f"Discord Webhook Adapter for Alertmanager")
    print("=" * 70)
    print(f"Listening on: http://0.0.0.0:{port}")
    print(f"Health check: http://localhost:{port}/")
    print(f"Discord webhook: {'✅ Configured' if DISCORD_WEBHOOK_URL else '❌ Not configured'}")
    print("=" * 70)
    print("Waiting for alerts from Alertmanager...")
    print()
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down...")
        server.shutdown()

if __name__ == '__main__':
    main()

