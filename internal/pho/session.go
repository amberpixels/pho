package pho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"pho/internal/hashing"
	"time"
)

const (
	phoSessionFile = "_session.json"
)

var (
	ErrNoSession = errors.New("no session found")
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

// SaveSession saves session metadata to the unified session.conf file.
func (app *App) SaveSession(_ context.Context, queryParams QueryParameters) error {
	if err := app.setupPhoDir(); err != nil {
		return fmt.Errorf("failed to setup pho directory: %w", err)
	}

	// Determine dump filename - use default if no renderer is available
	dumpFilename := phoDumpBase + ".jsonl" // Default to JSONL format
	if app.render != nil {
		dumpFilename = app.getDumpFilename()
	}

	sessionPath := filepath.Join(phoDir, phoSessionConf)
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

	return nil
}

// LoadSession loads session metadata from the .pho directory.
func (app *App) LoadSession(_ context.Context) (*SessionMetadata, error) {
	// Try to read from new session.conf format first
	sessionConfPath := filepath.Join(phoDir, phoSessionConf)
	if data, err := os.ReadFile(sessionConfPath); err == nil {
		sessionConfig := &SessionConfig{}
		if err := sessionConfig.FromSessionConf(data); err == nil {
			return sessionConfig.ToSessionMetadata(), nil
		}
	}

	// Fall back to legacy _session.json format
	sessionPath := filepath.Join(phoDir, phoSessionFile)
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return nil, ErrNoSession
	}

	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session SessionMetadata
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session metadata: %w", err)
	}

	return &session, nil
}

// ClearSession removes session metadata and associated files.
func (app *App) ClearSession(_ context.Context) error {
	// Remove new session.conf file if it exists
	sessionConfPath := filepath.Join(phoDir, phoSessionConf)
	if _, err := os.Stat(sessionConfPath); err == nil {
		if err := os.Remove(sessionConfPath); err != nil {
			return fmt.Errorf("failed to remove session config file: %w", err)
		}
	}

	// Remove legacy session file if it exists
	sessionPath := filepath.Join(phoDir, phoSessionFile)
	if _, err := os.Stat(sessionPath); err == nil {
		if err := os.Remove(sessionPath); err != nil {
			return fmt.Errorf("failed to remove legacy session file: %w", err)
		}
	}

	// Remove legacy meta file if it exists
	metaPath := filepath.Join(phoDir, phoMetaFile)
	if _, err := os.Stat(metaPath); err == nil {
		if err := os.Remove(metaPath); err != nil {
			return fmt.Errorf("failed to remove legacy meta file: %w", err)
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
		return false, nil, err
	}

	return true, session, nil
}

// ValidateSession checks if the session files still exist and are valid.
func (app *App) ValidateSession(_ context.Context, session *SessionMetadata) error {
	if session == nil {
		return errors.New("session is nil")
	}

	// Check if dump file exists
	dumpPath := filepath.Join(phoDir, session.DumpFile)
	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		return fmt.Errorf("session dump file missing: %s", session.DumpFile)
	}

	// Check if session.conf exists (new format) or meta file exists (legacy)
	sessionConfPath := filepath.Join(phoDir, phoSessionConf)
	metaPath := filepath.Join(phoDir, session.MetaFile)

	if _, err := os.Stat(sessionConfPath); err == nil {
		// New format exists, validation passed
		return nil
	}

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("session meta file missing: %s", session.MetaFile)
	}

	return nil
}
