package api

import (
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	"github.com/1broseidon/hallmonitor/internal/logging"
)

// isDevMode checks if we're running in development mode
func isDevMode() bool {
	return os.Getenv("HALLMONITOR_DEV") == "true" || os.Getenv("HALLMONITOR_DEV") == "1"
}

// getHTMLPath returns the path to HTML files (for dev mode)
func getHTMLPath(filename string) string {
	// Try to find the file relative to the project root
	// This works when running from the project root
	if _, err := os.Stat(filepath.Join("internal", "api", filename)); err == nil {
		return filepath.Join("internal", "api", filename)
	}
	// Fallback to current directory
	return filename
}

// dashboardHandler serves the embedded dashboard HTML or from disk in dev mode
func (s *Server) dashboardHandler(c *fiber.Ctx) error {
	// Check for view preference
	view := c.Query("view", "")
	if view == "" {
		// Check localStorage preference would be handled client-side
		// Default to ambient view (minimal, zen interface)
		view = "ambient"
	}

	// In dev mode, serve from disk for hot-reloading
	if isDevMode() {
		var filename string
		if view == "metric" {
			filename = "dashboard.html"
		} else {
			filename = "dashboard_ambient.html"
		}

		htmlPath := getHTMLPath(filename)
		if content, err := os.ReadFile(htmlPath); err == nil {
			return c.Type("html").SendString(string(content))
		}
		// Fall back to embedded if file not found
		s.logger.WithComponent(logging.ComponentAPI).
			WithFields(map[string]interface{}{
				"path": htmlPath,
			}).
			Warn("Dev mode: HTML file not found, falling back to embedded")
	}

	// Serve embedded HTML (production mode or fallback)
	if view == "metric" {
		return c.Type("html").SendString(dashboardHTML)
	}

	// Default to ambient view
	return c.Type("html").SendString(dashboardAmbientHTML)
}

// exportGrafanaDashboardHandler exports the Grafana dashboard JSON
func (s *Server) exportGrafanaDashboardHandler(c *fiber.Ctx) error {
	// Return a message directing users to create their own Grafana dashboard
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"message": "Grafana dashboard export is not available",
		"note":    "Please set up your own Grafana instance and connect it to Hall Monitor's /metrics endpoint",
		"docs":    "See documentation for Prometheus and Grafana setup instructions",
	})
}
