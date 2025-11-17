package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/1broseidon/hallmonitor/internal/config"
	"github.com/1broseidon/hallmonitor/internal/logging"
	"github.com/1broseidon/hallmonitor/pkg/models"
)

// MonitorCreateRequest represents a request to create a monitor
type MonitorCreateRequest struct {
	GroupName string         `json:"group_name"` // Which group to add the monitor to
	Monitor   models.Monitor `json:"monitor"`
}

// MonitorUpdateRequest represents a request to update a monitor
type MonitorUpdateRequest struct {
	Monitor models.Monitor `json:"monitor"`
}

// GroupCreateRequest represents a request to create a group
type GroupCreateRequest struct {
	Group models.MonitorGroup `json:"group"`
}

// GroupUpdateRequest represents a request to update a group
type GroupUpdateRequest struct {
	Group models.MonitorGroup `json:"group"`
}

// ConfigUpdateRequest represents a request to update the entire config
type ConfigUpdateRequest struct {
	Config config.Config `json:"config"`
}

// createMonitorHandler creates a new monitor
func (s *Server) createMonitorHandler(c *fiber.Ctx) error {
	var req MonitorCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	// Validate required fields
	if req.GroupName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "group_name is required",
		})
	}

	if req.Monitor.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "monitor name is required",
		})
	}

	// Load current config
	cfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to load config for monitor creation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to load configuration",
			"error":   err.Error(),
		})
	}

	// Add monitor to config
	if err := cfg.AddMonitor(req.GroupName, req.Monitor); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to add monitor",
			"error":   err.Error(),
		})
	}

	// Validate modified config
	if err := cfg.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Configuration validation failed",
			"error":   err.Error(),
		})
	}

	// Write config to file
	if err := cfg.WriteConfig(s.configPath); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to write config after monitor creation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to save configuration",
			"error":   err.Error(),
		})
	}

	// Reload configuration
	if err := s.ReloadConfig(c.Context()); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to reload config after monitor creation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Configuration saved but reload failed",
			"error":   err.Error(),
		})
	}

	s.logger.WithComponent(logging.ComponentAPI).
		WithFields(map[string]interface{}{
			"monitor": req.Monitor.Name,
			"group":   req.GroupName,
		}).
		Info("Monitor created successfully")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Monitor %s created successfully", req.Monitor.Name),
		"monitor": req.Monitor,
	})
}

// updateMonitorHandler updates an existing monitor
func (s *Server) updateMonitorHandler(c *fiber.Ctx) error {
	monitorName := c.Params("name")
	if monitorName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "monitor name is required",
		})
	}

	var req MonitorUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	// Load current config
	cfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to load config for monitor update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to load configuration",
			"error":   err.Error(),
		})
	}

	// Update monitor in config
	if err := cfg.UpdateMonitor(monitorName, req.Monitor); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update monitor",
			"error":   err.Error(),
		})
	}

	// Validate modified config
	if err := cfg.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Configuration validation failed",
			"error":   err.Error(),
		})
	}

	// Write config to file
	if err := cfg.WriteConfig(s.configPath); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to write config after monitor update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to save configuration",
			"error":   err.Error(),
		})
	}

	// Reload configuration
	if err := s.ReloadConfig(c.Context()); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to reload config after monitor update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Configuration saved but reload failed",
			"error":   err.Error(),
		})
	}

	s.logger.WithComponent(logging.ComponentAPI).
		WithFields(map[string]interface{}{
			"monitor": monitorName,
		}).
		Info("Monitor updated successfully")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Monitor %s updated successfully", monitorName),
		"monitor": req.Monitor,
	})
}

// deleteMonitorHandler deletes a monitor
func (s *Server) deleteMonitorHandler(c *fiber.Ctx) error {
	monitorName := c.Params("name")
	if monitorName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "monitor name is required",
		})
	}

	// Load current config
	cfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to load config for monitor deletion")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to load configuration",
			"error":   err.Error(),
		})
	}

	// Delete monitor from config
	if err := cfg.DeleteMonitor(monitorName); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete monitor",
			"error":   err.Error(),
		})
	}

	// Validate modified config
	if err := cfg.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Configuration validation failed",
			"error":   err.Error(),
		})
	}

	// Write config to file
	if err := cfg.WriteConfig(s.configPath); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to write config after monitor deletion")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to save configuration",
			"error":   err.Error(),
		})
	}

	// Reload configuration
	if err := s.ReloadConfig(c.Context()); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to reload config after monitor deletion")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Configuration saved but reload failed",
			"error":   err.Error(),
		})
	}

	s.logger.WithComponent(logging.ComponentAPI).
		WithFields(map[string]interface{}{
			"monitor": monitorName,
		}).
		Info("Monitor deleted successfully")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Monitor %s deleted successfully", monitorName),
	})
}

// createGroupHandler creates a new monitoring group
func (s *Server) createGroupHandler(c *fiber.Ctx) error {
	var req GroupCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	if req.Group.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "group name is required",
		})
	}

	// Load current config
	cfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to load config for group creation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to load configuration",
			"error":   err.Error(),
		})
	}

	// Add group to config
	if err := cfg.AddGroup(req.Group); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to add group",
			"error":   err.Error(),
		})
	}

	// Validate modified config
	if err := cfg.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Configuration validation failed",
			"error":   err.Error(),
		})
	}

	// Write config to file
	if err := cfg.WriteConfig(s.configPath); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to write config after group creation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to save configuration",
			"error":   err.Error(),
		})
	}

	// Reload configuration
	if err := s.ReloadConfig(c.Context()); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to reload config after group creation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Configuration saved but reload failed",
			"error":   err.Error(),
		})
	}

	s.logger.WithComponent(logging.ComponentAPI).
		WithFields(map[string]interface{}{
			"group": req.Group.Name,
		}).
		Info("Group created successfully")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Group %s created successfully", req.Group.Name),
		"group":   req.Group,
	})
}

// updateGroupHandler updates an existing group
func (s *Server) updateGroupHandler(c *fiber.Ctx) error {
	groupName := c.Params("name")
	if groupName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "group name is required",
		})
	}

	var req GroupUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	// Load current config
	cfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to load config for group update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to load configuration",
			"error":   err.Error(),
		})
	}

	// Update group in config
	if err := cfg.UpdateGroup(groupName, req.Group); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Failed to update group",
			"error":   err.Error(),
		})
	}

	// Validate modified config
	if err := cfg.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Configuration validation failed",
			"error":   err.Error(),
		})
	}

	// Write config to file
	if err := cfg.WriteConfig(s.configPath); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to write config after group update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to save configuration",
			"error":   err.Error(),
		})
	}

	// Reload configuration
	if err := s.ReloadConfig(c.Context()); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to reload config after group update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Configuration saved but reload failed",
			"error":   err.Error(),
		})
	}

	s.logger.WithComponent(logging.ComponentAPI).
		WithFields(map[string]interface{}{
			"group": groupName,
		}).
		Info("Group updated successfully")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Group %s updated successfully", groupName),
		"group":   req.Group,
	})
}

// deleteGroupHandler deletes a group
func (s *Server) deleteGroupHandler(c *fiber.Ctx) error {
	groupName := c.Params("name")
	if groupName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "group name is required",
		})
	}

	// Load current config
	cfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to load config for group deletion")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to load configuration",
			"error":   err.Error(),
		})
	}

	// Delete group from config
	if err := cfg.DeleteGroup(groupName); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Failed to delete group",
			"error":   err.Error(),
		})
	}

	// Validate modified config
	if err := cfg.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Configuration validation failed",
			"error":   err.Error(),
		})
	}

	// Write config to file
	if err := cfg.WriteConfig(s.configPath); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to write config after group deletion")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to save configuration",
			"error":   err.Error(),
		})
	}

	// Reload configuration
	if err := s.ReloadConfig(c.Context()); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to reload config after group deletion")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Configuration saved but reload failed",
			"error":   err.Error(),
		})
	}

	s.logger.WithComponent(logging.ComponentAPI).
		WithFields(map[string]interface{}{
			"group": groupName,
		}).
		Info("Group deleted successfully")

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Group %s deleted successfully", groupName),
	})
}

// updateConfigHandler updates the entire configuration
func (s *Server) updateConfigHandler(c *fiber.Ctx) error {
	var req ConfigUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request body",
			"error":   err.Error(),
		})
	}

	// Validate the new config
	if err := req.Config.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Configuration validation failed",
			"error":   err.Error(),
		})
	}

	// Write config to file
	if err := req.Config.WriteConfig(s.configPath); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to write config")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to save configuration",
			"error":   err.Error(),
		})
	}

	// Reload configuration
	if err := s.ReloadConfig(c.Context()); err != nil {
		s.logger.WithComponent(logging.ComponentAPI).
			WithError(err).
			Error("Failed to reload config after full update")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Configuration saved but reload failed",
			"error":   err.Error(),
		})
	}

	s.logger.WithComponent(logging.ComponentAPI).
		Info("Configuration updated successfully")

	return c.JSON(fiber.Map{
		"success":        true,
		"message":        "Configuration updated successfully",
		"total_monitors": len(s.monitorManager.GetMonitors()),
		"total_groups":   len(s.monitorManager.GetGroups()),
	})
}
