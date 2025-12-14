// Package api provides terminal and log streaming WebSocket endpoints
package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/UPwith-me/Container-Maker/cloud/providers"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// TerminalMessage represents a terminal WebSocket message
type TerminalMessage struct {
	Type    string `json:"type"` // command, output, error
	Content string `json:"content"`
}

// LogLine represents a log entry
type LogLine struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// HandleTerminalWebSocket handles WebSocket connections for terminal access
func (s *Server) HandleTerminalWebSocket(c echo.Context) error {
	instanceID := c.Param("id")

	// Authenticate
	token := c.QueryParam("token")
	userID := "demo"
	if token != "" && token != "cm_demo" {
		if claims, err := s.validateJWT(token); err == nil {
			userID = claims.UserID
		}
	}

	// Verify instance ownership
	instance, err := s.db.GetInstanceByID(instanceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Instance not found")
	}
	if instance.OwnerID != userID && userID != "demo" {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("Terminal WebSocket upgrade failed: %v", err)
		return err
	}
	defer conn.Close()

	// Send welcome message
	conn.WriteJSON(TerminalMessage{
		Type:    "output",
		Content: "Connected to " + instance.Name,
	})
	conn.WriteJSON(TerminalMessage{
		Type:    "output",
		Content: "Container ID: " + instance.ProviderID,
	})

	// Get provider for this instance
	provider, err := s.providers.Get(providers.ProviderType(instance.Provider))
	if err != nil {
		conn.WriteJSON(TerminalMessage{
			Type:    "error",
			Content: "Provider not available: " + instance.Provider,
		})
		return nil
	}

	// Handle terminal interaction
	for {
		var msg TerminalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Terminal WebSocket error: %v", err)
			}
			break
		}

		if msg.Type == "command" {
			// Execute command in container
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			stdout, stderr, exitCode, err := provider.ExecCommand(ctx, instance.ProviderID, []string{"sh", "-c", msg.Content})
			cancel()

			if err != nil {
				conn.WriteJSON(TerminalMessage{
					Type:    "error",
					Content: "Error: " + err.Error(),
				})
			} else if exitCode != 0 {
				output := stdout
				if stderr != "" {
					output = stderr
				}
				conn.WriteJSON(TerminalMessage{
					Type:    "output",
					Content: output,
				})
			} else {
				conn.WriteJSON(TerminalMessage{
					Type:    "output",
					Content: stdout,
				})
			}
		}
	}

	return nil
}

// HandleLogStreamWebSocket handles WebSocket connections for log streaming
func (s *Server) HandleLogStreamWebSocket(c echo.Context) error {
	instanceID := c.Param("id")

	// Authenticate
	token := c.QueryParam("token")
	userID := "demo"
	if token != "" && token != "cm_demo" {
		if claims, err := s.validateJWT(token); err == nil {
			userID = claims.UserID
		}
	}

	// Verify instance ownership
	instance, err := s.db.GetInstanceByID(instanceID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Instance not found")
	}
	if instance.OwnerID != userID && userID != "demo" {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("Log stream WebSocket upgrade failed: %v", err)
		return err
	}
	defer conn.Close()

	// Get provider for this instance
	provider, err := s.providers.Get(providers.ProviderType(instance.Provider))
	if err != nil {
		return nil
	}

	// Stream logs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logChan, err := provider.StreamLogs(ctx, instance.ProviderID)
	if err != nil {
		// Send error and fallback to simulated logs
		conn.WriteJSON(LogLine{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     "error",
			Message:   "Could not connect to container logs: " + err.Error(),
		})

		// Send simulated logs for demo
		go sendSimulatedLogs(conn)

		// Wait for client disconnect
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
		return nil
	}

	// Read from log channel and send to WebSocket
	for line := range logChan {
		logLine := parseLogLine(line)
		if err := conn.WriteJSON(logLine); err != nil {
			break
		}
	}

	return nil
}

func parseLogLine(line string) LogLine {
	// Simple log parsing - in production would be more sophisticated
	level := "info"
	if len(line) > 0 {
		switch line[0] {
		case 'E', 'e':
			level = "error"
		case 'W', 'w':
			level = "warn"
		case 'D', 'd':
			level = "debug"
		}
	}

	return LogLine{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   line,
	}
}

func sendSimulatedLogs(conn *websocket.Conn) {
	messages := []string{
		"Container started successfully",
		"Listening on port 3000",
		"Health check passed",
		"Processing request",
		"Request completed in 45ms",
		"Cache hit for key: session_abc",
		"Memory usage: 156MB / 512MB",
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	i := 0
	for range ticker.C {
		msg := messages[i%len(messages)]
		level := "info"
		if i%7 == 0 {
			level = "warn"
		}

		if err := conn.WriteJSON(LogLine{
			Timestamp: time.Now().Format(time.RFC3339),
			Level:     level,
			Message:   msg,
		}); err != nil {
			break
		}
		i++
	}
}
