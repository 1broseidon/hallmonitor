package api

import (
	"github.com/gofiber/fiber/v2"
)

// dashboardHandler serves the embedded dashboard HTML
func (s *Server) dashboardHandler(c *fiber.Ctx) error {
	// Check for view preference
	view := c.Query("view", "")
	if view == "" {
		// Check localStorage preference would be handled client-side
		// Default to ambient view (minimal, zen interface)
		view = "ambient"
	}

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
