# Pho Development Plan

## Project Vision
Transform Pho into a universal database document editor that supports multiple database backends (MongoDB, ElasticSearch, PostgreSQL) with a consistent interface for querying, editing, and applying changes.

## Current State Assessment

### ✅ **Working Features**
- Basic MongoDB connection and querying
- **Complete CRUD operations** (INSERT, UPDATE, DELETE, NOOP) ✅ **NEW**
- Document editing workflow (query → edit → diff → apply)
- **Context-aware file operations** with cancellation support ✅ **NEW**
- **Automatic file extension handling** (.json/.jsonl based on content) ✅ **NEW**
- JSONL parsing with comment support (92.9% test coverage)
- ExtJSON marshalling with stable output (66.7% test coverage)
- **Document hashing and change detection** (94.7% test coverage) ✅ **NEW**
- **Advanced diff engine** (100% test coverage) ✅ **NEW**
- **Go-idiomatic Action enum** with JSON serialization ✅ **NEW**
- MongoDB shell command generation
- **Data safety** with cloned operations (no mutation bugs) ✅ **NEW**
- Basic CLI interface

### ❌ **Critical Missing Features**
- ~~Missing MongoDB Shell ExtJSON v1 mode~~ ✅ **COMPLETED**
- ~~Database connection details not persisted in metadata~~ ✅ **COMPLETED**

### ⚠️ **Technical Debt**
- **Reduced TODO items** (most critical ones resolved) ✅ **IMPROVED**
- ~~Hardcoded configuration values~~ ✅ **IMPROVED** (ExtJSON modes now configurable)
- Inefficient round-trip marshalling in ExtJSON
- Basic CLI without proper help/shorthand flags ✅ **IMPROVED** (new configuration flags added)
- **Need more test coverage** for remaining packages (internal/pho, internal/render, internal/restore)

## Development Roadmap

### Phase 1: Foundation Stabilization (Priority: Critical)
**Timeline: 2-3 weeks**

#### 1.1 Fix Critical Bugs ✅ COMPLETED
- [x] **Fix data mutation issues** in restore operations (clone data objects)
- [x] **Complete CRUD implementations** (INSERT/DELETE operations)
- [x] **Implement context usage** in all file operations
- [x] **Add file extension automation** (.json/.jsonl based on content)

#### 1.2 Comprehensive Testing ✅ COMPLETED
- [x] **Unit tests for critical packages** (current: 61.6% overall coverage) ✅ **COMPLETED**
  - [x] `internal/diff/` - Change detection logic (100% coverage) ✅
  - [x] `internal/hashing/` - Document identification and checksums (94.7% coverage) ✅
  - [x] `internal/pho/` - Core application logic (43.9% coverage) ✅ **NEW**
  - [x] `internal/render/` - Output formatting (97.4% coverage) ✅ **NEW**
  - [x] `internal/restore/` - Change application (50.8% coverage) ✅ **NEW**
- [ ] **Integration tests** for end-to-end workflows
- [x] **Error handling tests** for edge cases ✅ **COMPLETED** (comprehensive coverage across all packages)

#### 1.3 MongoDB Shell ExtJSON v1 Support ✅ COMPLETED
- [x] **Implement ExtJSON v1 Shell mode** in renderer ✅
- [x] **Add configuration flags** for ExtJSON modes ✅
- [x] **Test compatibility** with MongoDB shell ✅
- [x] **Store database connection details** in metadata ✅

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
