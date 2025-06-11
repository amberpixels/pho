# Known Issues and Technical Debt

## 🎉 Recent Progress Summary

**Phase 1.1 Critical Bugs - COMPLETED**
- ✅ Fixed data mutation bugs in restore operations
- ✅ Implemented complete CRUD operations (INSERT/DELETE/UPDATE/NOOP)
- ✅ Added context usage to all file operations for proper cancellation
- ✅ Implemented automatic file extension (.json/.jsonl) based on content format

**Impact**: Core functionality is now stable and reliable. Data corruption risks eliminated.

## Critical Issues (Must Fix Before v1.0)

### ✅ **Data Corruption Risk** - RESOLVED
**Location**: `internal/restore/refactor_via_mongo_shell.go`, `internal/restore/refactor_via_mongo_client.go`
**Issue**: Data objects were being mutated instead of cloned during restore operations
**Solution**: Added `cloneBsonM()` helper function in `internal/restore/helpers.go` to create shallow copies of bson.M data before modification
**Status**: **COMPLETED** - All restore operations now use cloned data to prevent mutation

### ✅ **Incomplete CRUD Operations** - RESOLVED
**Location**: `internal/restore/refactor_via_mongo_client.go`
**Issue**: Only UPDATE operations were implemented, missing INSERT and DELETE
**Solution**: Implemented complete CRUD operations:
- **INSERT**: `insertOne()` with proper error handling
- **DELETE**: `deleteOne()` with verification of deletion count
- **UPDATE**: Enhanced with better error handling
- **NOOP**: Proper handling with `ErrNoop` return
**Status**: **COMPLETED** - Full CRUD functionality implemented

### ✅ **Missing ExtJSON v1 Shell Mode** - RESOLVED
**Location**: `internal/render/renderer.go`
**Issue**: MongoDB Shell ExtJSON v1 mode not implemented
**Solution**: Implemented complete ExtJSON v1 Shell mode with proper MongoDB constructors:
- `ObjectId("...")` instead of `{"$oid": "..."}`
- `ISODate("...")` instead of `{"$date": "..."}`
- `NumberLong("...")` instead of `{"$numberLong": "..."}`
- `NumberInt("...")` instead of `{"$numberInt": "..."}`
**Status**: **COMPLETED** - Full Shell mode compatibility achieved

## High Priority Issues

### ⚠️ **Zero Test Coverage on Core Logic**
**Locations**: All `internal/` packages
**Issue**: 0% test coverage on critical business logic
**Current Coverage**:
- `internal/pho`: 0%
- `internal/diff`: 0%
- `internal/hashing`: 0%
- `internal/render`: 0%
- `internal/restore`: 0%
**Impact**: No confidence in code reliability, difficult to refactor
**Priority**: HIGH
**Estimated Fix Time**: 40-60 hours

### ✅ **File Extension Confusion** - RESOLVED
**Location**: `internal/pho/app.go`
**Issue**: Always created `.jsonl` files regardless of content format
**Solution**: Implemented dynamic file extension system:
- Added `getDumpFileExtension()` method that determines extension based on renderer configuration
- `.json` extension for `AsValidJSON` mode (JSON array format)
- `.jsonl` extension for compact/default mode (line-by-line format)
- Updated `readDump()` to handle both JSON array and JSONL formats
**Status**: **COMPLETED** - File extensions now automatically match content format for proper editor syntax highlighting

### ✅ **Missing Database Connection Persistence** - RESOLVED
**Location**: `cmd/pho/main.go`, `internal/pho/model.go`, `internal/pho/app.go`
**Issue**: Connection details not stored in metadata for review/apply operations
**Solution**: Implemented JSON-based metadata storage with connection details:
- Enhanced `ParsedMeta` structure to include URI, Database, and Collection
- Added `ConnectDBForApply` method that uses stored metadata
- Backward compatibility with old line-based metadata format
- Connection details automatically persisted during dump operations
**Status**: **COMPLETED** - Apply operations now work without specifying connection details

### ✅ **No Context Usage in File Operations** - RESOLVED
**Location**: `internal/pho/app.go`
**Issue**: File operations didn't use context for cancellation/timeout
**Solution**: Updated all file operations to use context:
- `readMeta(ctx)` and `readDump(ctx)` now accept context parameters
- Added context cancellation checks in file reading loops
- `extractChanges(ctx)` passes context through the call chain
**Status**: **COMPLETED** - All file operations now support cancellation and timeout

## Medium Priority Issues

### 🔧 **Poor CLI Experience** ✅ **IMPROVED**
**Locations**: `cmd/pho/main.go`
**Issues**: 
- ~~No proper help text with examples~~ ✅ **IMPROVED** (comprehensive help added)
- ~~No shorthand flags~~ ✅ **IMPROVED** (added -m, -c, -l, -e flags)
- ~~Basic flag parsing without validation~~ ✅ **IMPROVED** (added ExtJSON mode validation)
**New Features Added**:
- `--extjson-mode` / `-m` for ExtJSON mode selection (canonical/relaxed/shell)
- `--compact` / `-c` for compact JSON output
- `--line-numbers` / `-l` for line number control
- Comprehensive help text with flag descriptions
**Impact**: Significantly improved user experience and discoverability
**Priority**: MEDIUM
**Status**: **IMPROVED** - Core CLI functionality enhanced, Kong integration remains for future

### 🔧 **Configuration Inflexibility** ✅ **IMPROVED**
**Locations**: Multiple files
**Issues**:
- ~~ExtJSON mode hardcoded~~ ✅ **FIXED** (now configurable via flags)
- No verbosity control (`internal/pho/app.go:412`)
- No configuration file support
**Improvements Made**:
- ExtJSON mode now configurable with `--extjson-mode` flag
- Added compact JSON and line numbers configuration
- Input validation for ExtJSON mode values
**Impact**: Significantly improved customization options
**Priority**: MEDIUM
**Status**: **IMPROVED** - Core configuration flexibility added, verbosity and config files remain for future

### 🔧 **Performance Issues**
**Location**: `pkg/extjson/extjson.go:61-62`
**Issue**: Inefficient round-trip marshalling
```go
// TODO(2): rewrite so it's a efficient solution
```
**Impact**: Poor performance on large documents
**Priority**: MEDIUM
**Estimated Fix Time**: 6-8 hours

## Low Priority Issues

### 📝 **Code Organization**
**Locations**: Various files
**Issues**:
- Hardcoded values that should be configurable
- Missing utility functions (should be in helpers)
- Enum declarations could be richer
**Impact**: Code maintainability
**Priority**: LOW
**Estimated Fix Time**: 4-6 hours

### 📝 **Error Handling Improvements**
**Location**: `internal/pho/app.go:139`
**Issue**: Hash file creation failure should be a warning, not a hard error
```go
// TODO: it should be a soft error (warning)
//       so we still dump data, but not letting to edit it
```
**Impact**: Application crashes on non-critical errors
**Priority**: LOW
**Estimated Fix Time**: 1-2 hours

## Architectural Debt

### 🏗️ **ExtJSON Architecture Inconsistency**
**Locations**: `internal/render/renderer.go`, `pkg/extjson/extjson.go`
**Issue**: Three different ExtJSON implementations create architectural inconsistency:
1. **Direct `bson.MarshalExtJSON`** calls in renderer (Canonical/Relaxed modes)
2. **`pkg/extjson`** package with stable marshalling (unused in renderer)
3. **`marshalShellExtJSON`** custom implementation (Shell mode)

**Problems**:
- Canonical/Relaxed modes don't use stable key ordering
- Code duplication and inconsistent patterns
- Different logic paths for different ExtJSON modes
- `pkg/extjson` package is orphaned

**Proposed Solution**:
```go
// Unified ExtJSON interface in pkg/extjson
type Marshaller interface {
    Marshal(v any) ([]byte, error)
}

// Implementations:
- CanonicalMarshaller (stable bson.MarshalExtJSON wrapper)
- RelaxedMarshaller (stable bson.MarshalExtJSON wrapper)
- ShellMarshaller (current marshalShellExtJSON logic)

// Renderer uses unified interface:
marshaller := extjson.NewMarshaller(cfg.ExtJSONMode)
result := marshaller.Marshal(document)
```

**Benefits**:
- Consistent stable marshalling across all modes
- Single code path for all ExtJSON handling
- Better maintainability and extensibility
- Proper separation of concerns

**Impact**: Improved code maintainability and consistency
**Priority**: MEDIUM
**Estimated Fix Time**: 6-8 hours

### 🏗️ **Tight Coupling to MongoDB**
**Issue**: All code is tightly coupled to MongoDB-specific types and operations
**Impact**: Difficult to add support for other databases
**Priority**: MEDIUM (for multi-DB goal)
**Estimated Fix Time**: 20-30 hours (major refactoring)

### 🏗️ **No Plugin Architecture**
**Issue**: No way to extend functionality without modifying core code
**Impact**: Limited extensibility
**Priority**: LOW
**Estimated Fix Time**: 15-20 hours

### 🏗️ **Limited Error Context**
**Issue**: Errors don't provide enough context for debugging
**Impact**: Difficult to troubleshoot issues
**Priority**: LOW
**Estimated Fix Time**: 4-6 hours

## Incomplete Features (TODOs)

### 📋 **Placeholder TODOs**
**Locations**: `internal/pho/model.go:11`, `model.go:14`
**Issues**: Empty TODOs without clear requirements
```go
// TODO:
// ExtJSON params
//
// TODO:
//dbName     string
//collection string
```
**Impact**: Unclear feature requirements
**Priority**: LOW
**Action**: Need requirements clarification

### 📋 **Editor Integration Improvements**
**Location**: `internal/pho/app.go:250-262`
**Issue**: Basic editor spawning without editor-specific optimizations
**Impact**: Suboptimal editing experience
**Priority**: LOW
**Estimated Fix Time**: 3-4 hours

## Security Concerns

### 🔒 **Temporary File Security**
**Issue**: `.pho/` files created with default permissions
**Impact**: Potentially sensitive data accessible to other users
**Priority**: MEDIUM
**Fix**: Use 0600 permissions for temporary files

### 🔒 **Connection String Exposure**
**Issue**: MongoDB URIs may contain credentials in plain text
**Impact**: Credential exposure in logs/process lists
**Priority**: MEDIUM
**Fix**: Mask credentials in logging and error messages

## Performance Concerns

### ⚡ **Memory Usage**
**Issue**: All documents loaded into memory simultaneously
**Impact**: Cannot handle very large result sets
**Priority**: MEDIUM
**Fix**: Implement streaming/pagination

### ⚡ **No Connection Pooling**
**Issue**: Single connection per operation
**Impact**: Poor performance for bulk operations
**Priority**: LOW
**Fix**: Implement connection pooling

## Issue Resolution Strategy

### Immediate Actions (Next Sprint)
1. Fix data mutation bugs (2 hours)
2. Implement missing CRUD operations (6 hours)
3. Add basic unit tests for critical paths (16 hours)
4. Fix file extension handling (3 hours)

### Short Term (Next Month)
1. Complete test coverage to 80%
2. Implement Kong CLI integration
3. Add ExtJSON v1 Shell mode
4. Store connection details in metadata

### Medium Term (2-3 Months)
1. Database abstraction layer
2. Performance optimizations
3. Security improvements
4. Plugin architecture

### Tracking
- Create GitHub issues for each critical and high priority item
- Use GitHub Projects to track progress
- Set up CI/CD to prevent regression of fixed issues

## Risk Assessment

### **High Risk Issues**
- Data mutation bugs (could cause data loss)
- Missing CRUD operations (core functionality incomplete)
- Zero test coverage (high probability of bugs)

### **Medium Risk Issues**
- File extension confusion (user experience)
- Performance issues (scalability)
- Security concerns (data exposure)

### **Low Risk Issues**
- Code organization (maintainability)
- Incomplete features (nice-to-have functionality)

This issues list should be revisited regularly and updated as fixes are implemented and new issues are discovered.