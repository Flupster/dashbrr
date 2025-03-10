// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package maintainerr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/dashbrr/internal/models"
	"github.com/autobrr/dashbrr/internal/services/core"
)

// Custom error types for better error handling
type ErrMaintainerr struct {
	Op       string // Operation that failed
	Err      error  // Underlying error
	HttpCode int    // HTTP status code if applicable
}

func (e *ErrMaintainerr) Error() string {
	if e.HttpCode > 0 {
		return fmt.Sprintf("maintainerr %s: server returned %s (%d)", e.Op, http.StatusText(e.HttpCode), e.HttpCode)
	}
	if e.Err != nil {
		return fmt.Sprintf("maintainerr %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("maintainerr %s", e.Op)
}

func (e *ErrMaintainerr) Unwrap() error {
	return e.Err
}

type MaintainerrService struct {
	core.ServiceCore
}

type StatusResponse struct {
	Version         string `json:"version"`
	UpdateAvailable bool   `json:"updateAvailable"`
}

type Media struct {
	ID           int       `json:"id"`
	CollectionID int       `json:"collectionId"`
	PlexID       int       `json:"plexId"`
	TmdbID       int       `json:"tmdbId"`
	AddDate      time.Time `json:"addDate"`
	ImagePath    string    `json:"image_path"`
	IsManual     bool      `json:"isManual"`
}

type Collection struct {
	ID                int     `json:"id"`
	PlexID            int     `json:"plexId"`
	LibraryID         int     `json:"libraryId"`
	Title             string  `json:"title"`
	Description       string  `json:"description"`
	IsActive          bool    `json:"isActive"`
	ArrAction         int     `json:"arrAction"`
	VisibleOnHome     bool    `json:"visibleOnHome"`
	DeleteAfterDays   int     `json:"deleteAfterDays"`
	ManualCollection  bool    `json:"manualCollection"`
	ListExclusions    bool    `json:"listExclusions"`
	ForceOverseerr    bool    `json:"forceOverseerr"`
	Type              int     `json:"type"`
	KeepLogsForMonths int     `json:"keepLogsForMonths"`
	AddDate           string  `json:"addDate"`
	Media             []Media `json:"media"`
}

func init() {
	models.NewMaintainerrService = NewMaintainerrService
}

func NewMaintainerrService() models.ServiceHealthChecker {
	service := &MaintainerrService{}
	service.Type = "maintainerr"
	service.DisplayName = "Maintainerr"
	service.Description = "Monitor and manage your Maintainerr instance"
	service.DefaultURL = "http://localhost:6246"
	service.HealthEndpoint = "/api/app/status"
	return service
}

func (s *MaintainerrService) GetHealthEndpoint(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	return fmt.Sprintf("%s/api/app/status", baseURL)
}

func (s *MaintainerrService) getVersion(ctx context.Context, url string) (string, error) {
	if version := s.GetVersionFromCache(url); version != "" {
		return version, nil
	}

	healthEndpoint := s.GetHealthEndpoint(url)
	resp, err := s.MakeRequestWithContext(ctx, healthEndpoint, "", nil)
	if err != nil {
		return "", &ErrMaintainerr{Op: "get_version", Err: fmt.Errorf("failed to make request: %w", err)}
	}
	defer resp.Body.Close()

	body, err := s.ReadBody(resp)
	if err != nil {
		return "", &ErrMaintainerr{Op: "get_version", Err: fmt.Errorf("failed to read response: %w", err)}
	}

	var statusResponse StatusResponse
	if err := json.Unmarshal(body, &statusResponse); err != nil {
		return "", &ErrMaintainerr{Op: "get_version", Err: fmt.Errorf("failed to parse response: %w", err)}
	}

	if err := s.CacheVersion(url, statusResponse.Version, time.Hour); err != nil {
		// Log but don't fail if caching fails
		fmt.Printf("Failed to cache version: %v\n", err)
	}

	return statusResponse.Version, nil
}

func (s *MaintainerrService) CheckHealth(url, apiKey string) (models.ServiceHealth, int) {
	startTime := time.Now()

	if url == "" {
		return s.CreateHealthResponse(startTime, "error", "URL is required"), http.StatusBadRequest
	}

	ctx := context.Background()

	versionChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		version, err := s.getVersion(ctx, url)
		if err != nil {
			errChan <- err
			versionChan <- ""
			return
		}
		versionChan <- version
		errChan <- nil
	}()

	healthEndpoint := s.GetHealthEndpoint(url)
	resp, err := s.MakeRequestWithContext(ctx, healthEndpoint, "", nil)
	if err != nil {
		return s.CreateHealthResponse(startTime, "offline", fmt.Sprintf("Failed to connect: %v", err)), http.StatusOK
	}
	defer resp.Body.Close()

	responseTime, _ := time.ParseDuration(resp.Header.Get("X-Response-Time") + "ms")

	body, err := s.ReadBody(resp)
	if err != nil {
		return s.CreateHealthResponse(startTime, "error", fmt.Sprintf("Failed to read response: %v", err)), http.StatusOK
	}

	if resp.StatusCode >= 400 {
		statusText := http.StatusText(resp.StatusCode)
		status := "error"
		message := fmt.Sprintf("Server returned %s (%d)", statusText, resp.StatusCode)

		// Determine appropriate status based on response code
		switch resp.StatusCode {
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			message = fmt.Sprintf("Service is temporarily unavailable (%d %s)", resp.StatusCode, statusText)
		case http.StatusUnauthorized:
			message = "Invalid API key"
		case http.StatusForbidden:
			message = "Access forbidden"
		case http.StatusNotFound:
			message = "Service endpoint not found"
		}

		return s.CreateHealthResponse(startTime, status, message), http.StatusOK
	}

	var statusResponse StatusResponse
	if err := json.Unmarshal(body, &statusResponse); err != nil {
		return s.CreateHealthResponse(startTime, "error", fmt.Sprintf("Failed to parse status response: %v", err)), http.StatusOK
	}

	var version string
	var versionErr error
	select {
	case version = <-versionChan:
		versionErr = <-errChan
	case <-time.After(2 * time.Second):
		versionErr = fmt.Errorf("version check timed out")
	}

	extras := map[string]interface{}{
		"version":         version,
		"updateAvailable": statusResponse.UpdateAvailable,
		"responseTime":    responseTime.Milliseconds(),
	}

	if versionErr != nil {
		extras["versionError"] = versionErr.Error()
	}

	return s.CreateHealthResponse(startTime, "online", "Healthy", extras), http.StatusOK
}

func (s *MaintainerrService) GetCollections(url, apiKey string) ([]Collection, error) {
	if url == "" {
		return nil, &ErrMaintainerr{Op: "get_collections", Err: fmt.Errorf("URL is required")}
	}

	if apiKey == "" {
		return nil, &ErrMaintainerr{Op: "get_collections", Err: fmt.Errorf("API key is required")}
	}

	ctx := context.Background()

	baseURL := strings.TrimRight(url, "/")
	endpoint := fmt.Sprintf("%s/api/collections", baseURL)

	resp, err := s.MakeRequestWithContext(ctx, endpoint, apiKey, nil)
	if err != nil {
		return nil, &ErrMaintainerr{Op: "get_collections", Err: fmt.Errorf("failed to connect: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &ErrMaintainerr{
			Op:       "get_collections",
			HttpCode: resp.StatusCode,
		}
	}

	body, err := s.ReadBody(resp)
	if err != nil {
		return nil, &ErrMaintainerr{Op: "get_collections", Err: fmt.Errorf("failed to read response: %w", err)}
	}

	var collections []Collection
	if err := json.Unmarshal(body, &collections); err != nil {
		// Try parsing as single collection if array parse fails
		var singleCollection Collection
		if err := json.Unmarshal(body, &singleCollection); err != nil {
			return nil, &ErrMaintainerr{Op: "get_collections", Err: fmt.Errorf("failed to parse response: %w", err)}
		}
		if singleCollection.IsActive {
			collections = []Collection{singleCollection}
		} else {
			collections = []Collection{}
		}
	}

	activeCollections := make([]Collection, 0)
	for _, collection := range collections {
		if collection.IsActive {
			activeCollections = append(activeCollections, collection)
		}
	}

	return activeCollections, nil
}
