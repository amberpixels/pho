# Pho Architecture Documentation

## Current Architecture Overview

Pho is designed as a MongoDB document editor with a workflow-based architecture that supports querying, editing, and applying changes back to the database.

### High-Level Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI Layer     │    │   Core App       │    │  Database       │
│  (cmd/pho)      │────│ (internal/pho)   │────│  (MongoDB)      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ├─── Renderer (internal/render)
                              ├─── Diff Engine (internal/diff)
                              ├─── Hashing (internal/hashing)
                              └─── Restore (internal/restore)
```

### Component Architecture

#### 1. CLI Layer (`cmd/pho/`)
**Responsibility**: Command-line interface and flag parsing
- Entry point for the application
- Flag parsing and validation
- URI construction and connection parameters
- Workflow orchestration (query → edit → review → apply)

**Current Capabilities** ✅ **IMPROVED**:
- Comprehensive flag parsing with help text
- Shorthand flags (-m, -c, -l, -e)
- ExtJSON mode validation and configuration
- Connection parameter handling
- Configuration flags for rendering options

**Enhanced CLI Flags** ✅ **NEW**:
- `--extjson-mode` / `-m`: ExtJSON format (canonical/relaxed/shell)
- `--compact` / `-c`: Compact JSON output
- `--line-numbers` / `-l`: Line number control
- `--edit` / `-e`: Editor configuration

#### 2. Core Application (`internal/pho/`)
**Responsibility**: Main application logic and MongoDB integration
- MongoDB connection management
- Query execution and cursor handling
- File system operations (`.pho/` directory)
- Editor integration
- Workflow coordination

**Key Components**:
- `App` struct - Main application controller
- Connection handling with context support
- Temporary file management
- Editor spawning and interaction

#### 3. Document Rendering (`internal/render/`)
**Responsibility**: Format MongoDB documents for display and editing
- ExtJSON formatting (canonical, relaxed, shell modes)
- Line number generation
- Configuration management
- Format validation

**Current Capabilities**:
- Canonical and Relaxed ExtJSON v2
- Line numbering for editors
- Configurable output formats

**Current Capabilities**:
- Canonical, Relaxed, and Shell ExtJSON formats ✅ **NEW**
- ExtJSON v1 Shell mode with MongoDB constructors ✅ **NEW**
- Line numbering for editors
- Configurable output formats
- CLI configuration flags ✅ **NEW**

**Remaining Missing Features**:
- Automatic format detection
- Custom formatting options

#### 4. Change Detection (`internal/diff/`)
**Responsibility**: Detect and categorize document changes
- Document comparison via hashing
- Change categorization (ADDED, UPDATED, DELETED, NOOP)
- Change filtering and processing
- Bulk change operations

**Core Algorithm**:
1. Hash original documents during dump
2. Hash edited documents after editing
3. Compare hashes to detect changes
4. Generate change objects with actions

#### 5. Document Hashing (`internal/hashing/`)
**Responsibility**: Generate unique identifiers and checksums for documents
- Document identification via `_id` field
- SHA256 checksumming for change detection
- Hash parsing and serialization
- Identifier value handling (ObjectID, strings)

**Hash Format**: `{identifier_field}::{identifier_value}|{checksum}`
**Example**: `_id::507f1f77bcf86cd799439011|a1b2c3d4e5f6...`

#### 6. Change Restoration (`internal/restore/`)
**Responsibility**: Apply changes back to MongoDB
- Two restoration strategies:
  - **MongoClient**: Direct execution via Go driver
  - **MongoShell**: Command generation for manual execution
- CRUD operation handling
- Error handling and rollback

**Current Limitations**:
- Only UPDATE operations fully implemented
- Missing INSERT and DELETE operations
- No transaction support
- Data mutation bugs (needs cloning)

#### 7. Utility Packages (`pkg/`)
**JSONL Parser** (`pkg/jsonl/`):
- Line-by-line JSON parsing
- Comment support in JSONL files
- Robust error handling
- 92.9% test coverage

**ExtJSON Utilities** (`pkg/extjson/`):
- Stable ExtJSON marshalling
- MongoDB BSON compatibility
- 66.7% test coverage

**⚠️ ExtJSON Architecture Issue**:
Currently there are **three different ExtJSON implementations** in the codebase:
1. `bson.MarshalExtJSON` (MongoDB driver) - used directly in renderer
2. `pkg/extjson` package - provides stable marshalling but unused in renderer
3. `marshalShellExtJSON` - custom Shell mode implementation

This creates inconsistency where Canonical/Relaxed modes don't use stable marshalling, while Shell mode has completely separate logic. Future refactoring should unify all ExtJSON handling through the `pkg/extjson` package.

## Data Flow Architecture

### 1. Query Phase
```
User Input → CLI Parsing → App.ConnectDB() → App.RunQuery() → MongoDB Cursor
```

### 2. Dump Phase
```
MongoDB Cursor → App.Dump() → Document Hashing → File Writing
                      ↓
                 .pho/_dump.jsonl + .pho/_meta
```

### 3. Edit Phase
```
.pho/_dump.jsonl → External Editor → Modified Documents
```

### 4. Review Phase
```
Modified Documents → Change Detection → Diff Calculation → Review Output
                          ↓
                    Hash Comparison + Change Categorization
```

### 5. Apply Phase
```
Changes → Restore Strategy → MongoDB Operations → Database Updates
```

## File System Architecture

### `.pho/` Directory Structure
```
.pho/
├── _meta          # JSON metadata with connection details and document hashes
└── _dump.jsonl    # Editable document dump
```

**Enhanced Metadata Format** (`_meta`) ✅ **NEW**:
```json
{
  "URI": "mongodb://localhost:27017",
  "Database": "mydb",
  "Collection": "mycoll",
  "Lines": {
    "_id::507f1f77bcf86cd799439011|a1b2c3d4e5f6789abcdef123456": {
      "IdentifiedBy": "_id",
      "IdentifierValue": "507f1f77bcf86cd799439011",
      "Checksum": "a1b2c3d4e5f6789abcdef123456"
    }
  }
}
```

**Legacy Metadata Format** (backward compatible):
```
_id::507f1f77bcf86cd799439011|a1b2c3d4e5f6789abcdef123456
_id::507f1f77bcf86cd799439012|b2c3d4e5f6789abcdef123456789
```

**Dump Format** (`_dump.jsonl`):
```json
{"_id": {"$oid": "507f1f77bcf86cd799439011"}, "name": "Document 1"}
{"_id": {"$oid": "507f1f77bcf86cd799439012"}, "name": "Document 2"}
```

## Extension Points for Multi-Database Support

### Proposed Database Abstraction Layer

```go
// Core interfaces for database abstraction
type DatabaseDriver interface {
    Connect(ctx context.Context, config ConnectionConfig) error
    Query(ctx context.Context, query QuerySpec) (DocumentCursor, error)
    Apply(ctx context.Context, changes []Change) error
    Close(ctx context.Context) error
}

type DocumentCursor interface {
    Next(ctx context.Context) bool
    Decode(interface{}) error
    Close(ctx context.Context) error
}

type QuerySpec struct {
    Database   string
    Collection string
    Filter     map[string]interface{}
    Sort       map[string]interface{}
    Projection map[string]interface{}
    Limit      int64
}
```

### Driver Registry System

```go
type DriverRegistry struct {
    drivers map[string]DatabaseDriver
}

func (r *DriverRegistry) Register(name string, driver DatabaseDriver)
func (r *DriverRegistry) Get(name string) (DatabaseDriver, error)
func (r *DriverRegistry) List() []string
```

### Database-Specific Implementations

#### MongoDB Driver
- Current functionality wrapped in driver interface
- Extended JSON handling
- BSON document support
- GridFS support for large documents

#### ElasticSearch Driver
- HTTP-based connection
- JSON DSL query translation
- Index and type management
- Bulk operations support

#### PostgreSQL Driver
- SQL-based operations
- JSON/JSONB column support
- Row-based document model
- Transaction support

## Configuration Architecture

### Current Configuration
- Command-line flags only
- Hardcoded defaults
- No persistence

### Proposed Configuration System

```yaml
# ~/.pho/config.yaml
default_driver: mongodb
editor: vim
format: jsonl

drivers:
  mongodb:
    uri: mongodb://localhost:27017
    timeout: 30s
  
  elasticsearch:
    url: http://localhost:9200
    auth: basic
    
  postgresql:
    host: localhost
    port: 5432
    database: documents
```

## Error Handling Architecture

### Current Error Handling
- Basic error wrapping
- Limited context information
- No recovery mechanisms

### Proposed Error Handling

```go
type PhoError struct {
    Code      ErrorCode
    Message   string
    Context   map[string]interface{}
    Cause     error
    Timestamp time.Time
}

type ErrorCode int

const (
    ErrConnection ErrorCode = iota
    ErrQuery
    ErrParsing
    ErrFileSystem
    ErrEditor
    ErrRestore
)
```

## Testing Architecture

### Current Testing
- Only `pkg/` packages have tests
- No integration tests
- No mocking framework

### Proposed Testing Strategy

```
tests/
├── unit/           # Package-level unit tests
├── integration/    # End-to-end workflow tests
├── fixtures/       # Test data and configurations
└── mocks/          # Generated mocks for interfaces
```

**Testing Layers**:
1. **Unit Tests**: Individual component testing
2. **Integration Tests**: Database driver testing
3. **E2E Tests**: Complete workflow testing
4. **Performance Tests**: Large dataset handling

## Security Architecture

### Current Security
- No authentication handling
- Connection strings in plain text
- No encryption for temporary files

### Proposed Security Enhancements
- Credential management via environment variables
- Encrypted temporary file storage
- Connection string masking in logs
- Audit logging for database operations

## Performance Architecture

### Current Performance Characteristics
- Synchronous operations
- In-memory document storage
- No pagination for large result sets
- Round-trip marshalling inefficiency

### Proposed Performance Optimizations
- Streaming document processing
- Cursor-based pagination
- Background processing for large operations
- Memory-mapped file handling
- Connection pooling

This architecture provides a solid foundation for extending Pho into a multi-database document editor while maintaining clean separation of concerns and extensibility.