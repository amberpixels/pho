package pho

import (
	"bufio"
	"encoding/json"
	"fmt"
	"pho/internal/hashing"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// ParsedMeta stores hashed lines and other meta.
type ParsedMeta struct {
	// Connection details for review/apply operations
	URI        string
	Database   string
	Collection string

	// Lines are hashes per identifier.
	// Identifier here is considered to be identified_by field + identifier value
	// etc. _id::111111
	Lines map[string]*hashing.HashData
}

type DumpDoc bson.M

// UnmarshalJSON for now is a hack, as we hardcode the way unmarshal parameters in here
// Whole thing of DumpDoc is required, so it's properly parsed back from ExtJson into bson.
func (tx *DumpDoc) UnmarshalJSON(raw []byte) error {
	return bson.UnmarshalExtJSON(raw, true, tx)
}

// ToJSON serializes the metadata to JSON format.
func (meta *ParsedMeta) ToJSON() ([]byte, error) {
	return json.MarshalIndent(meta, "", "  ")
}

// FromJSON deserializes metadata from JSON format.
func (meta *ParsedMeta) FromJSON(data []byte) error {
	return json.Unmarshal(data, meta)
}

// SessionConfig represents the new unified session configuration.
type SessionConfig struct {
	// RFC 822 frontmatter fields
	Created       time.Time `conf:"Created"`
	URI           string    `conf:"URI"`
	Database      string    `conf:"Database"`
	Collection    string    `conf:"Collection"`
	Query         string    `conf:"Query"`
	Limit         int64     `conf:"Limit"`
	Sort          string    `conf:"Sort,omitempty"`
	Projection    string    `conf:"Projection,omitempty"`
	DumpFile      string    `conf:"DumpFile"`
	DocumentCount int       `conf:"DocumentCount"`

	// Hash data (key:value pairs in body)
	Lines map[string]*hashing.HashData
}

// ToSessionConf serializes the session config to RFC 822 + key:value format.
func (sc *SessionConfig) ToSessionConf() ([]byte, error) {
	var result strings.Builder

	// RFC 822 frontmatter
	result.WriteString(fmt.Sprintf("Created: %s\n", sc.Created.Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("URI: %s\n", sc.URI))
	result.WriteString(fmt.Sprintf("Database: %s\n", sc.Database))
	result.WriteString(fmt.Sprintf("Collection: %s\n", sc.Collection))
	result.WriteString(fmt.Sprintf("Query: %s\n", sc.Query))
	result.WriteString(fmt.Sprintf("Limit: %d\n", sc.Limit))

	if sc.Sort != "" {
		result.WriteString(fmt.Sprintf("Sort: %s\n", sc.Sort))
	}
	if sc.Projection != "" {
		result.WriteString(fmt.Sprintf("Projection: %s\n", sc.Projection))
	}

	result.WriteString(fmt.Sprintf("DumpFile: %s\n", sc.DumpFile))
	result.WriteString(fmt.Sprintf("DocumentCount: %d\n", sc.DocumentCount))

	// Empty line to separate frontmatter from body
	result.WriteString("\n")

	// Hash data as lines (one per document)
	for _, hashData := range sc.Lines {
		line := hashData.String()
		result.WriteString(fmt.Sprintf("%s\n", line))
	}

	return []byte(result.String()), nil
}

// FromSessionConf deserializes session config from RFC 822 + key:value format.
func (sc *SessionConfig) FromSessionConf(data []byte) error {
	reader := strings.NewReader(string(data))
	scanner := bufio.NewScanner(reader)

	// Initialize Lines map
	sc.Lines = make(map[string]*hashing.HashData)

	inFrontmatter := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Empty line separates frontmatter from body
		if line == "" && inFrontmatter {
			inFrontmatter = false
			continue
		}

		// Skip empty lines in body
		if line == "" {
			continue
		}

		if inFrontmatter {
			// Parse frontmatter as key: value
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Parse frontmatter fields
			if err := sc.parseFrontmatterField(key, value); err != nil {
				return fmt.Errorf("failed to parse frontmatter field %s: %w", key, err)
			}
		} else {
			// Parse hash data directly from line (format: _id::ObjectID(...)|checksum)
			hashData, err := hashing.Parse(line)
			if err != nil {
				return fmt.Errorf("failed to parse hash data: %w", err)
			}
			identifier := hashData.GetIdentifier()
			sc.Lines[identifier] = hashData
		}
	}

	return scanner.Err()
}

// parseFrontmatterField parses a single frontmatter field.
func (sc *SessionConfig) parseFrontmatterField(key, value string) error {
	switch key {
	case "Created":
		created, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return err
		}
		sc.Created = created
	case "URI":
		sc.URI = value
	case "Database":
		sc.Database = value
	case "Collection":
		sc.Collection = value
	case "Query":
		sc.Query = value
	case "Limit":
		limit, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		sc.Limit = limit
	case "Sort":
		sc.Sort = value
	case "Projection":
		sc.Projection = value
	case "DumpFile":
		sc.DumpFile = value
	case "DocumentCount":
		count, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		sc.DocumentCount = count
	}
	return nil
}

// ToSessionMetadata converts SessionConfig to SessionMetadata for backward compatibility.
func (sc *SessionConfig) ToSessionMetadata() *SessionMetadata {
	return &SessionMetadata{
		Created: sc.Created,
		QueryParams: QueryParameters{
			URI:        sc.URI,
			Database:   sc.Database,
			Collection: sc.Collection,
			Query:      sc.Query,
			Limit:      sc.Limit,
			Sort:       sc.Sort,
			Projection: sc.Projection,
		},
		DumpFile:      sc.DumpFile,
		MetaFile:      phoMetaFile, // Legacy field
		DocumentCount: sc.DocumentCount,
	}
}

// ToParsedMeta converts SessionConfig to ParsedMeta for backward compatibility.
func (sc *SessionConfig) ToParsedMeta() *ParsedMeta {
	return &ParsedMeta{
		URI:        sc.URI,
		Database:   sc.Database,
		Collection: sc.Collection,
		Lines:      sc.Lines,
	}
}

// FromSessionMetadataAndParsedMeta creates SessionConfig from old format.
func (sc *SessionConfig) FromSessionMetadataAndParsedMeta(session *SessionMetadata, meta *ParsedMeta) {
	sc.Created = session.Created
	sc.URI = session.QueryParams.URI
	sc.Database = session.QueryParams.Database
	sc.Collection = session.QueryParams.Collection
	sc.Query = session.QueryParams.Query
	sc.Limit = session.QueryParams.Limit
	sc.Sort = session.QueryParams.Sort
	sc.Projection = session.QueryParams.Projection
	sc.DumpFile = session.DumpFile
	sc.DocumentCount = session.DocumentCount
	sc.Lines = meta.Lines
}
