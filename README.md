# *pho* 🍜 — inline database editor.

<p align="center">
  <img src="logo.png" alt="Logo" width="300"/><br>
</p>


> Edit MongoDB documents right in your favorite editor

⚠️ **UNSTABLE**: This project is under active development

Pho bridges the gap between MongoDB and your preferred text editor. Query documents, edit them with your familiar tools and hotkeys, then apply changes back to your database.

## Why Pho?

- **Editor Freedom**: Use Neovim, VS Code, Sublime, Zed, or any editor of your choice
- **Familiar Workflow**: Leverage your existing macros, snippets, and muscle memory
- **Safe Editing**: Review changes before applying them to your database
- **Batch Operations**: Edit multiple documents simultaneously with powerful text manipulation

## Quick Start

```bash
# Install
go install github.com/amberpixels/pho/cmd/pho@latest

# Edit documents
pho --db myapp --collection users --query '{"active": true}' --edit nvim

# Review your changes
pho review

# Apply changes to database
pho apply
```

## How It Works

1. **Query** → Fetch documents matching your criteria
2. **Edit** → Documents open in your editor as JSON/JSONL
3. **Review** → See exactly what will change
4. **Apply** → Push changes back to MongoDB

```bash
# Query active users and edit in Neovim
pho --db shop --collection users --query '{"status": "active"}' --edit nvim

# Bulk update with your editor's power
# - Use find/replace across all documents
# - Apply consistent formatting
# - Leverage your custom snippets
```

## Key Features

- **Smart Change Detection**: Only modified documents are updated
- **Multiple Formats**: Work with canonical, relaxed, or shell-compatible JSON
- **Session Management**: Resume editing sessions across multiple commands
- **Environment Support**: Configure connection via environment variables
- **Verbose Logging**: Track operations with `--verbose` flag

## Installation

```bash
# Via Go
go install github.com/amberpixels/pho/cmd/pho@latest

# Build from source
git clone https://github.com/amberpixels/pho
cd pho
make install
```

## Examples

```bash
# Quick data fix
pho --uri mongodb://localhost:27017 --db shop --collection products --edit code

# Complex query with projection
pho --db analytics --collection events \
    --query '{"timestamp": {"$gte": "2024-01-01"}}' \
    --projection '{"_id": 1, "data": 1}' \
    --edit nvim

# Environment-based connection
export MONGODB_URI="mongodb://localhost:27017"
export MONGODB_DB="myapp"
pho --collection sessions --query '{}' --limit 50
```

## Coming Soon

- Support for PostgreSQL, MySQL, and other databases
- Advanced query builders
- Change previews with syntax highlighting

---

**Pho** • Edit databases like text files • [Report Issues](https://github.com/amberpixels/pho/issues)
