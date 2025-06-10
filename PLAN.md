# Pho Development Plan

## Project Vision
Transform Pho into a universal database document editor that supports multiple database backends (MongoDB, ElasticSearch, PostgreSQL) with a consistent interface for querying, editing, and applying changes.

## Current State Assessment

### ✅ **Working Features**
- Basic MongoDB connection and querying
- Document editing workflow (query → edit → diff → apply)
- JSONL parsing with comment support (92.9% test coverage)
- ExtJSON marshalling with stable output (66.7% test coverage)
- Document hashing and change detection
- MongoDB shell command generation
- Basic CLI interface

### ❌ **Critical Missing Features**
- **No unit tests** for core functionality (0% coverage on all internal packages)
- **Incomplete CRUD operations** (only UPDATE implemented, missing INSERT/DELETE)
- **Missing MongoDB Shell ExtJSON v1 mode**
- **File extension automation** (.json vs .jsonl for editor syntax highlighting)
- **Database connection details not persisted** in metadata
- **Data mutation bugs** (objects modified instead of cloned)

### ⚠️ **Technical Debt**
- 31 TODO items across 12 files
- Hardcoded configuration values
- Missing context usage in file operations
- Inefficient round-trip marshalling in ExtJSON
- Basic CLI without proper help/shorthand flags

## Development Roadmap

### Phase 1: Foundation Stabilization (Priority: Critical)
**Timeline: 2-3 weeks**

#### 1.1 Fix Critical Bugs ✅ COMPLETED
- [x] **Fix data mutation issues** in restore operations (clone data objects)
- [x] **Complete CRUD implementations** (INSERT/DELETE operations)  
- [x] **Implement context usage** in all file operations
- [x] **Add file extension automation** (.json/.jsonl based on content)

#### 1.2 Comprehensive Testing
- [ ] **Unit tests for all internal packages** (target: 80%+ coverage)
  - `internal/diff/` - Change detection logic
  - `internal/hashing/` - Document identification and checksums
  - `internal/pho/` - Core application logic
  - `internal/render/` - Output formatting
  - `internal/restore/` - Change application
- [ ] **Integration tests** for end-to-end workflows
- [ ] **Error handling tests** for edge cases

#### 1.3 MongoDB Shell ExtJSON v1 Support
- [ ] **Implement ExtJSON v1 Shell mode** in renderer
- [ ] **Add configuration flags** for ExtJSON modes
- [ ] **Test compatibility** with MongoDB shell

### Phase 2: CLI Enhancement (Priority: High)
**Timeline: 1-2 weeks**

#### 2.1 Kong Integration
- [ ] **Replace flag package with Kong** for better CLI experience
- [ ] **Add proper help text** with examples and flag descriptions
- [ ] **Implement shorthand flags** (-q, -h, -d, -c, etc.)
- [ ] **Add configuration file support** (YAML/JSON config)
- [ ] **Validation and error messages** for invalid inputs

#### 2.2 User Experience Improvements
- [ ] **Verbosity control** (--verbose, --quiet flags)
- [ ] **Progress indicators** for long operations
- [ ] **Better error messages** with suggestions
- [ ] **Connection testing** before queries

### Phase 3: Architecture Refactoring (Priority: High)
**Timeline: 3-4 weeks**

#### 3.1 Database Abstraction Layer
- [ ] **Design database interface** for multi-backend support
```go
type DatabaseDriver interface {
    Connect(ctx context.Context, config ConnectionConfig) error
    Query(ctx context.Context, query Query) (DocumentCursor, error)
    Apply(ctx context.Context, changes Changes) error
    Close(ctx context.Context) error
}
```

#### 3.2 Driver Plugin System
- [ ] **MongoDB driver implementation** (current functionality)
- [ ] **Plugin discovery and loading** mechanism
- [ ] **Driver configuration** and registration
- [ ] **Common document interface** across drivers

#### 3.3 Metadata Enhancement
- [ ] **Store connection details** in metadata for review/apply operations
- [ ] **Schema information** for better editing experience
- [ ] **Operation history** tracking
- [ ] **Metadata versioning** for backwards compatibility

### Phase 4: Multi-Database Support (Priority: Medium)
**Timeline: 4-6 weeks**

#### 4.1 ElasticSearch Driver
- [ ] **Connection handling** (HTTP-based)
- [ ] **Query translation** (JSON DSL ↔ Pho query format)
- [ ] **Document mapping** (Elasticsearch documents ↔ common format)
- [ ] **Index management** operations
- [ ] **Bulk operations** support

#### 4.2 PostgreSQL Driver
- [ ] **Connection handling** (SQL-based)
- [ ] **JSON/JSONB support** for document-like operations
- [ ] **Query translation** (SQL ↔ Pho query format)
- [ ] **Transaction support** for atomic changes
- [ ] **Schema awareness** for structured data

#### 4.3 Common Query Language
- [ ] **Design unified query syntax** that translates to each backend
- [ ] **Query validation** and error handling
- [ ] **Query optimization** hints
- [ ] **Cross-database compatibility** guidelines

### Phase 5: Advanced Features (Priority: Low)
**Timeline: 2-3 weeks**

#### 5.1 Enhanced Editing Experience
- [ ] **Syntax validation** in editor
- [ ] **Auto-completion** for field names
- [ ] **Diff visualization** in terminal
- [ ] **Undo/redo** functionality
- [ ] **Batch operations** support

#### 5.2 Performance Optimizations
- [ ] **Streaming operations** for large datasets
- [ ] **Parallel processing** for bulk operations
- [ ] **Caching layer** for repeated queries
- [ ] **Memory usage optimization**

#### 5.3 Advanced Configuration
- [ ] **Custom editor configurations** per file type
- [ ] **Template system** for common operations
- [ ] **Plugin system** for custom transformations
- [ ] **Export/import** configurations

## Implementation Priorities

### **IMMEDIATE (Week 1-2)**
1. Fix data mutation bugs
2. Complete CRUD operations
3. Add comprehensive unit tests
4. Implement Kong CLI

### **SHORT TERM (Month 1)**
1. MongoDB ExtJSON v1 support
2. File extension automation
3. Context usage in file operations
4. Enhanced error handling

### **MEDIUM TERM (Month 2-3)**
1. Database abstraction layer
2. Plugin system design
3. ElasticSearch driver
4. PostgreSQL driver

### **LONG TERM (Month 4+)**
1. Advanced editing features
2. Performance optimizations
3. Documentation and tutorials
4. Community contributions

## Quality Assurance Strategy

### Testing Requirements
- **Unit Tests**: 80%+ coverage for all packages
- **Integration Tests**: End-to-end workflow testing
- **Performance Tests**: Large dataset handling
- **Compatibility Tests**: Multiple database versions

### Code Quality
- **Linting**: golangci-lint with strict rules
- **Code Review**: All changes reviewed
- **Documentation**: GoDoc for all public APIs
- **Examples**: Working examples for each driver

### Release Process
- **Semantic Versioning**: Clear version strategy
- **Changelog**: Detailed release notes
- **Binary Releases**: Multi-platform support
- **Package Managers**: Homebrew, apt, etc.

## Success Metrics

### Technical Metrics
- **Test Coverage**: >80% for all packages
- **Performance**: Handle 100k+ documents efficiently
- **Memory Usage**: <100MB for typical operations
- **Startup Time**: <1 second

### User Experience Metrics
- **Setup Time**: <5 minutes from install to first use
- **Learning Curve**: Basic operations within 10 minutes
- **Error Recovery**: Clear error messages and solutions
- **Documentation**: Complete examples for all features

## Resource Requirements

### Development Time
- **Phase 1-2**: ~6 weeks (foundation + CLI)
- **Phase 3**: ~4 weeks (architecture)
- **Phase 4**: ~6 weeks (multi-DB support)
- **Phase 5**: ~3 weeks (advanced features)
- **Total**: ~5 months for complete implementation

### Dependencies
- **Kong**: CLI framework
- **Additional DB drivers**: elasticsearch, pq
- **Testing frameworks**: testify, gomock
- **Documentation**: cobra/doc, godoc

This plan provides a structured approach to evolving Pho into a production-ready, multi-database document editor while maintaining code quality and user experience standards.