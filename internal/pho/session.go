package pho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	phoSessionFile = "_session.json"
)

var (
	ErrNoSession = errors.New("no session found")
)

// SessionStatus represents the current state of a session.
type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "active"
	SessionStatusModified SessionStatus = "modified"
	SessionStatusReady    SessionStatus = "ready"
)

// SessionMetadata contains information about the current editing session.
type SessionMetadata struct {
	// Session identification
	Created time.Time     `json:"created"`
	Status  SessionStatus `json:"status"`

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

// SaveSession saves session metadata to the .pho directory.
func (app *App) SaveSession(_ context.Context, queryParams QueryParameters) error {
	if err := app.setupPhoDir(); err != nil {
		return fmt.Errorf("failed to setup pho directory: %w", err)
	}

	// Determine dump filename - use default if no renderer is available
	dumpFilename := phoDumpBase + ".jsonl" // Default to JSONL format
	if app.render != nil {
		dumpFilename = app.getDumpFilename()
	}

	session := &SessionMetadata{
		Created:     time.Now(),
		Status:      SessionStatusActive,
		QueryParams: queryParams,
		DumpFile:    dumpFilename,
		MetaFile:    phoMetaFile,
	}

	sessionPath := filepath.Join(phoDir, phoSessionFile)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session metadata: %w", err)
	}

	if err := os.WriteFile(sessionPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// LoadSession loads session metadata from the .pho directory.
func (app *App) LoadSession(_ context.Context) (*SessionMetadata, error) {
	sessionPath := filepath.Join(phoDir, phoSessionFile)

	// Check if session file exists
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
	sessionPath := filepath.Join(phoDir, phoSessionFile)

	// Remove session file if it exists
	if _, err := os.Stat(sessionPath); err == nil {
		if err := os.Remove(sessionPath); err != nil {
			return fmt.Errorf("failed to remove session file: %w", err)
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

// UpdateSessionStatus updates the status of the current session.
func (app *App) UpdateSessionStatus(ctx context.Context, status SessionStatus) error {
	session, err := app.LoadSession(ctx)
	if err != nil {
		if errors.Is(err, ErrNoSession) {
			return errors.New("no active session found")
		}
		return err
	}

	session.Status = status

	sessionPath := filepath.Join(phoDir, phoSessionFile)
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session metadata: %w", err)
	}

	if err := os.WriteFile(sessionPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
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

	// Check if meta file exists
	metaPath := filepath.Join(phoDir, session.MetaFile)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("session meta file missing: %s", session.MetaFile)
	}

	return nil
}
