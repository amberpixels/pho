package pho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"pho/internal/hashing"
	"strings"
	"time"
)

var (
	ErrNoSession   = errors.New("no session found")
	ErrSessionLost = errors.New("session data lost")
)

// SessionMetadata contains information about the current editing session.
type SessionMetadata struct {
	// Session identification
	Created time.Time `json:"created"`

	// Query parameters that created this session
	QueryParams QueryParameters `json:"query_params"`

	// File information
	DumpFile      string `json:"dump_file"`
	MetaFile      string `json:"meta_file"`
	DocumentCount int    `json:"document_count"`
}

// QueryParameters stores the original query information.
type QueryParameters struct {
	URI        string `json:"uri"`
	Database   string `json:"database"`
	Collection string `json:"collection"`
	Query      string `json:"query"`
	Limit      int64  `json:"limit"`
	Sort       string `json:"sort,omitempty"`
	Projection string `json:"projection,omitempty"`
}

// String returns a human-readable description of the session.
func (s *SessionMetadata) String() string {
	return fmt.Sprintf("Session: %s.%s, Query: %s, Created: %s",
		s.QueryParams.Database,
		s.QueryParams.Collection,
		s.QueryParams.Query,
		s.Created.Format("2006-01-02 15:04:05"))
}

// Age returns how long ago the session was created.
func (s *SessionMetadata) Age() time.Duration {
	return time.Since(s.Created)
}

// SessionRegistry represents a lightweight session entry in the registry.
type SessionRegistry struct {
	ID            string          `json:"id"`
	Created       time.Time       `json:"created"`
	DataPath      string          `json:"data_path"`
	QueryParams   QueryParameters `json:"query_params"`
	DumpFile      string          `json:"dump_file"`
	DocumentCount int             `json:"document_count"`
}

// SessionStatus represents the current state of a session.
type SessionStatus int

const (
	SessionStatusActive   SessionStatus = iota // Registry exists and data files exist
	SessionStatusLost                          // Registry exists but data files are missing
	SessionStatusNotFound                      // No registry entry found
)

// getCurrentSessionID returns the current session ID (for now, just "current").
func getCurrentSessionID() string {
	return "current"
}

// getSessionRegistryPath returns the path to the session registry file.
func getSessionRegistryPath(sessionID string) (string, error) {
	sessionsDir, err := getSessionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(sessionsDir, sessionID+".json"), nil
}

// loadSessionRegistry loads session registry from file.
func loadSessionRegistry(sessionID string) (*SessionRegistry, error) {
	registryPath, err := getSessionRegistryPath(sessionID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSession
		}
		return nil, fmt.Errorf("failed to read session registry: %w", err)
	}

	var registry SessionRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session registry: %w", err)
	}

	return &registry, nil
}

// saveSessionRegistry saves session registry to file.
func (app *App) saveSessionRegistry(registry *SessionRegistry) error {
	if err := app.setupSessionsDir(); err != nil {
		return err
	}

	registryPath, err := getSessionRegistryPath(registry.ID)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session registry: %w", err)
	}

	if err := os.WriteFile(registryPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session registry: %w", err)
	}

	return nil
}

// checkSessionStatus checks the status of a session.
func checkSessionStatus(registry *SessionRegistry) SessionStatus {
	// Check if dump file exists
	dumpPath := filepath.Join(registry.DataPath, registry.DumpFile)
	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		return SessionStatusLost
	}

	// Check if session.conf exists
	sessionConfPath := filepath.Join(registry.DataPath, phoSessionConf)
	if _, err := os.Stat(sessionConfPath); err == nil {
		return SessionStatusActive
	}

	// session.conf doesn't exist
	return SessionStatusLost
}

// GetSessionStatus returns the current session status with diagnostic information.
func (app *App) GetSessionStatus(_ context.Context) (SessionStatus, *SessionRegistry, error) {
	sessionID := getCurrentSessionID()

	// Try to load session registry
	registry, err := loadSessionRegistry(sessionID)
	if err != nil {
		if errors.Is(err, ErrNoSession) {
			return SessionStatusNotFound, nil, nil
		}
		return SessionStatusNotFound, nil, fmt.Errorf("failed to load session registry: %w", err)
	}

	// Check if data files exist
	status := checkSessionStatus(registry)
	return status, registry, nil
}

// RecoverSession attempts to recover a lost session by re-running the query.
func (app *App) RecoverSession(ctx context.Context) (*SessionMetadata, error) {
	sessionID := getCurrentSessionID()

	// Load session registry to get the original query parameters
	registry, err := loadSessionRegistry(sessionID)
	if err != nil {
		if errors.Is(err, ErrNoSession) {
			return nil, fmt.Errorf("no session to recover: %w", ErrNoSession)
		}
		return nil, fmt.Errorf("failed to load session registry for recovery: %w", err)
	}

	// Clear the lost session data
	if err := app.ClearSession(ctx); err != nil {
		return nil, fmt.Errorf("failed to clear lost session: %w", err)
	}

	// Re-save the session with the original parameters
	if err := app.SaveSession(ctx, registry.QueryParams); err != nil {
		return nil, fmt.Errorf("failed to recreate session: %w", err)
	}

	// Return the recovered session metadata
	return app.LoadSession(ctx)
}

// ListSessions returns all session registries.
func (app *App) ListSessions(_ context.Context) ([]*SessionRegistry, error) {
	sessionsDir, err := getSessionsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions directory: %w", err)
	}

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*SessionRegistry{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []*SessionRegistry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".json")
		registry, err := loadSessionRegistry(sessionID)
		if err != nil {
			// Skip invalid registry files
			continue
		}

		sessions = append(sessions, registry)
	}

	return sessions, nil
}

// CleanupStaleSessions removes registry entries for sessions with missing data files.
func (app *App) CleanupStaleSessions(ctx context.Context) error {
	sessions, err := app.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	for _, session := range sessions {
		if checkSessionStatus(session) == SessionStatusLost {
			registryPath, err := getSessionRegistryPath(session.ID)
			if err != nil {
				continue
			}

			if err := os.Remove(registryPath); err != nil {
				return fmt.Errorf("failed to remove stale session registry %s: %w", session.ID, err)
			}
		}
	}

	return nil
}

// SaveSession saves session metadata to the unified session.conf file and session registry.
func (app *App) SaveSession(_ context.Context, queryParams QueryParameters) error {
	if err := app.setupPhoDir(); err != nil {
		return fmt.Errorf("failed to setup pho directory: %w", err)
	}

	// Determine dump filename - use default if no renderer is available
	dumpFilename := phoDumpBase + ".jsonl" // Default to JSONL format
	if app.render != nil {
		dumpFilename = app.getDumpFilename()
	}

	// Get the data directory path
	dataDir, err := getPhoDataDir()
	if err != nil {
		return fmt.Errorf("failed to get pho data dir: %w", err)
	}

	sessionPath := filepath.Join(dataDir, phoSessionConf)
	sessionConfig := &SessionConfig{
		Created:       time.Now(),
		URI:           queryParams.URI,
		Database:      queryParams.Database,
		Collection:    queryParams.Collection,
		Query:         queryParams.Query,
		Limit:         queryParams.Limit,
		Sort:          queryParams.Sort,
		Projection:    queryParams.Projection,
		DumpFile:      dumpFilename,
		DocumentCount: 0, // Will be updated when metadata is written
		Lines:         make(map[string]*hashing.HashData),
	}

	// If session.conf already exists, preserve the DocumentCount and Lines that were set by writeMetadata
	if data, err := os.ReadFile(sessionPath); err == nil {
		existingConfig := &SessionConfig{}
		if err := existingConfig.FromSessionConf(data); err == nil {
			// Preserve metadata that was already written
			sessionConfig.DocumentCount = existingConfig.DocumentCount
			sessionConfig.Lines = existingConfig.Lines
		}
	}

	data, err := sessionConfig.ToSessionConf()
	if err != nil {
		return fmt.Errorf("failed to serialize session config: %w", err)
	}

	if err := os.WriteFile(sessionPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session config file: %w", err)
	}

	// Create session registry entry
	sessionID := getCurrentSessionID()
	registry := &SessionRegistry{
		ID:            sessionID,
		Created:       sessionConfig.Created,
		DataPath:      dataDir,
		QueryParams:   queryParams,
		DumpFile:      dumpFilename,
		DocumentCount: sessionConfig.DocumentCount,
	}

	// Save session registry
	if err := app.saveSessionRegistry(registry); err != nil {
		return fmt.Errorf("failed to save session registry: %w", err)
	}

	return nil
}

// LoadSession loads session metadata, checking registry first and handling session loss.
func (app *App) LoadSession(ctx context.Context) (*SessionMetadata, error) {
	// Check session status first
	status, registry, err := app.GetSessionStatus(ctx)
	if err != nil {
		return nil, err
	}

	switch status {
	case SessionStatusNotFound:
		return nil, ErrNoSession

	case SessionStatusLost:
		// Registry exists but data files are missing
		return nil, fmt.Errorf("%w: session created %s ago, data files missing from %s",
			ErrSessionLost,
			time.Since(registry.Created).Round(time.Minute),
			registry.DataPath)

	case SessionStatusActive:
		// Load from data directory
		dataDir := registry.DataPath

		// Read from session.conf format
		sessionConfPath := filepath.Join(dataDir, phoSessionConf)
		data, err := os.ReadFile(sessionConfPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read session config file: %w", err)
		}

		sessionConfig := &SessionConfig{}
		if err := sessionConfig.FromSessionConf(data); err != nil {
			return nil, fmt.Errorf("failed to parse session config: %w", err)
		}

		return sessionConfig.ToSessionMetadata(), nil

	default:
		return nil, fmt.Errorf("unknown session status: %d", status)
	}
}

// ClearSession removes session metadata and associated files.
func (app *App) ClearSession(_ context.Context) error {
	sessionID := getCurrentSessionID()

	// Remove session registry
	registryPath, err := getSessionRegistryPath(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session registry path: %w", err)
	}

	if _, err := os.Stat(registryPath); err == nil {
		if err := os.Remove(registryPath); err != nil {
			return fmt.Errorf("failed to remove session registry: %w", err)
		}
	}

	// Get data directory
	dataDir, err := getPhoDataDir()
	if err != nil {
		return fmt.Errorf("failed to get pho data dir: %w", err)
	}

	// Remove session files from data directory
	sessionConfPath := filepath.Join(dataDir, phoSessionConf)
	if _, err := os.Stat(sessionConfPath); err == nil {
		if err := os.Remove(sessionConfPath); err != nil {
			return fmt.Errorf("failed to remove session config file: %w", err)
		}
	}

	return nil
}

// HasActiveSession checks if there's an active session.
func (app *App) HasActiveSession(ctx context.Context) (bool, *SessionMetadata, error) {
	session, err := app.LoadSession(ctx)
	if err != nil {
		if errors.Is(err, ErrNoSession) {
			return false, nil, nil
		}
		if errors.Is(err, ErrSessionLost) {
			// Session registry exists but data is lost - return specific error
			return false, nil, err
		}
		return false, nil, err
	}

	return true, session, nil
}

// ValidateSession checks if the session files still exist and are valid.
func (app *App) ValidateSession(_ context.Context, session *SessionMetadata) error {
	if session == nil {
		return errors.New("session is nil")
	}

	// Get data directory path
	dataDir, err := getPhoDataDir()
	if err != nil {
		return fmt.Errorf("could not get pho data dir: %w", err)
	}

	// Check if dump file exists
	dumpPath := filepath.Join(dataDir, session.DumpFile)
	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		return fmt.Errorf("session dump file missing: %s", session.DumpFile)
	}

	// Check if session.conf exists
	sessionConfPath := filepath.Join(dataDir, phoSessionConf)
	if _, err := os.Stat(sessionConfPath); os.IsNotExist(err) {
		return fmt.Errorf("session config file missing: %s", phoSessionConf)
	}

	return nil
}
