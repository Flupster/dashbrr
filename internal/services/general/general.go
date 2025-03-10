// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package general

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/dashbrr/internal/models"
	"github.com/autobrr/dashbrr/internal/services/core"
)

func init() {
	models.NewGeneralService = NewGeneralService
}

func NewGeneralService() models.ServiceHealthChecker {
	service := &GeneralService{}
	service.Type = "general"
	service.DisplayName = "" // Allow display name to be set via configuration
	service.Description = "Generic health check service for any URL endpoint"
	return service
}

type GeneralService struct {
	core.ServiceCore
}

func (s *GeneralService) CheckHealth(url, apiKey string) (models.ServiceHealth, int) {
	startTime := time.Now()

	if url == "" {
		return s.CreateHealthResponse(startTime, "error", "URL is required"), http.StatusBadRequest
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	headers := make(map[string]string)
	if apiKey != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", apiKey)
	}

	resp, err := s.MakeRequestWithContext(ctx, url, apiKey, headers)
	if err != nil {
		return s.CreateHealthResponse(startTime, "offline", fmt.Sprintf("Failed to connect: %v", err)), http.StatusServiceUnavailable
	}
	defer resp.Body.Close()

	responseTime, _ := time.ParseDuration(resp.Header.Get("X-Response-Time") + "ms")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return s.CreateHealthResponse(startTime, "error", fmt.Sprintf("Failed to read response: %v", err)), http.StatusInternalServerError
	}

	// Try to parse as JSON first
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err == nil {
		// Handle JSON response
		status := "online"
		message := ""

		if statusVal, ok := jsonResponse["status"].(string); ok {
			// Map status values to our supported statuses
			switch strings.ToLower(statusVal) {
			case "healthy", "ok", "online":
				status = "online"
			case "unhealthy", "error", "offline":
				status = "offline"
			case "warning":
				status = "warning"
			default:
				status = "unknown"
			}
		}
		if messageVal, ok := jsonResponse["message"].(string); ok {
			message = messageVal
		}

		extras := map[string]interface{}{
			"responseTime": responseTime.Milliseconds(),
		}

		return s.CreateHealthResponse(startTime, status, message, extras), resp.StatusCode
	}

	// If JSON parsing fails, treat as plain text
	textResponse := strings.TrimSpace(string(body))
	extras := map[string]interface{}{
		"responseTime": responseTime.Milliseconds(),
	}

	if strings.EqualFold(textResponse, "ok") {
		return s.CreateHealthResponse(startTime, "online", "", extras), resp.StatusCode
	}

	return s.CreateHealthResponse(startTime, "error", fmt.Sprintf("Unexpected response: %s", textResponse), extras), resp.StatusCode
}

func (s *GeneralService) GetVersion(url, apiKey string) (string, error) {
	return "", nil // Version not supported for general service
}

func (s *GeneralService) GetLatestVersion() (string, error) {
	return "", nil // Version not supported for general service
}
