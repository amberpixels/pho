# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

**Build**: `make build` - Builds the binary to `build/pho`
**Run**: `make run` - Builds and runs the application
**Test**: `make test` - Runs all tests with verbose output
**Lint**: `make lint` - Runs golangci-lint (installs if needed)
**Format**: `make tidy` - Formats code, runs vet, and tidies modules
**Install**: `make install` - Installs pho globally to GOPATH/bin

## Architecture Overview

Pho is a MongoDB document editor that allows querying, editing, and applying changes back to MongoDB collections through an interactive editor workflow.

### Core Workflow
1. **Query** - Connect to MongoDB and query documents with filters/projections
2. **Edit** - Dump documents to temporary files and open in editor (vim, etc.)
3. **Diff** - Compare original vs edited documents to detect changes
4. **Apply** - Execute changes back to MongoDB or generate shell commands

### Key Packages

**internal/pho/** - Main application logic and MongoDB connection handling
- `App` struct manages the complete workflow from query to change application
- Handles editor launching, temporary file management, and MongoDB operations
- Context-aware file operations with proper cancellation support
- Automatic file extension handling (.json/.jsonl) based on output format

**internal/diff/** - Change detection system
- Compares document hashes to identify additions, updates, deletions, and no-ops
- `CalculateChanges()` is the core function that computes document differences

**internal/hashing/** - Document identification and checksumming
- Creates unique identifiers for documents in format `_id::123|checksum`
- Uses SHA256 for content hashing and supports ObjectID/string identifiers

**internal/render/** - Output formatting
- Renders MongoDB documents in Extended JSON formats (canonical, relaxed, shell)
- Configurable line numbers, compaction, and validation settings

**internal/restore/** - Change application ✅ Recently Enhanced
- Two strategies: direct MongoDB client execution or shell command generation
- **Complete CRUD operations**: INSERT, UPDATE, DELETE, and NOOP handling
- **Data safety**: All operations use cloned data to prevent mutation
- **Error handling**: Proper validation and meaningful error messages

**pkg/jsonl/** - JSONL parsing with comment support
**pkg/extjson/** - Extended JSON utilities for MongoDB document handling

### File Structure ✅ Recently Enhanced
- `.pho/_meta` - Stores document hashes for change detection
- `.pho/_dump.json` or `.pho/_dump.jsonl` - Temporary file containing editable documents
  - **Automatic extension**: `.json` for valid JSON arrays, `.jsonl` for line-by-line format
  - **Editor support**: Proper syntax highlighting based on file extension
- Context-aware file operations with cancellation support

### MongoDB Connection
Supports standard MongoDB connection patterns similar to mongodump:
- `--uri` for connection strings
- `--host`/`--port` for individual components
- `--db` and `--collection` for targeting specific data