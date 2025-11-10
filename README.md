# ArmyKnife CLI - Semantic Code Search & Intelligence

[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Latest Release](https://img.shields.io/github/v/release/armyknifelabs-platform/armyknife-cli?style=flat-square&logo=github&color=brightgreen)](https://github.com/armyknifelabs-platform/armyknife-cli/releases)
[![Release Downloads](https://img.shields.io/github/downloads/armyknifelabs-platform/armyknife-cli/latest/total?style=flat-square&logo=github)](https://github.com/armyknifelabs-platform/armyknife-cli/releases)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/armyknifelabs-platform/armyknife-cli)](https://goreportcard.com/report/github.com/armyknifelabs-platform/armyknife-cli)
[![Code Intelligence](https://img.shields.io/badge/Code-Intelligence-blue?style=flat-square&logo=github)](https://github.com/armyknifelabs-platform/codeinsight-ai)

ArmyKnife is a powerful command-line tool for semantic code search and intelligence across your organization's repositories. Ask natural language questions about your codebase and get intelligent results powered by AI embeddings.

## Features

✨ **Semantic Code Search** - Ask questions in natural language, not just keywords
🔍 **Multi-Repository Search** - Search across multiple repositories simultaneously
⚡ **Hybrid Search** - Combine vector similarity with keyword matching for best results
🚀 **Enterprise-Ready** - Designed for large codebases with thousands of files
📊 **Performance Monitoring** - Built-in metrics for query latency and cache efficiency
🛠️ **Multiple Languages** - Supports TypeScript, Go, Python, Rust, Java, C/C++, Ruby, PHP, and more

## Quick Start

### Installation

#### From GitHub Releases (Recommended)

```bash
# Download the latest release for your platform
wget https://github.com/armyknifelabs-platform/armyknife-cli/releases/download/v1.0.0/armyknife-v1.0.0-linux-amd64
chmod +x armyknife-v1.0.0-linux-amd64
sudo mv armyknife-v1.0.0-linux-amd64 /usr/local/bin/armyknife

# Verify installation
armyknife --version
```

#### From Homebrew (macOS/Linux)

```bash
brew install armyknifelabs/tap/armyknife
```

#### From Source

```bash
git clone https://github.com/armyknifelabs-platform/armyknife-cli
cd armyknife-cli
go build -o armyknife ./cmd
sudo mv armyknife /usr/local/bin/
```

### Configuration

Set your Code Intelligence API endpoint:

```bash
# For local development (default)
export CODE_API_URL="http://localhost:3001/api/v1"

# For production
export CODE_API_URL="https://code-intelligence.armyknifelabs.com/api/v1"
export CODE_API_KEY="your_api_key_here"
```

### First Command

```bash
# Query your indexed code
armyknife code query "How does authentication work?"

# See performance metrics
armyknife code metrics

# View index statistics
armyknife code stats
```

## Usage

### Index a Repository

Index a codebase for semantic searching:

```bash
armyknife code index /path/to/repo --repo-id 1
```

**Options:**
- `--repo-id` (required): Unique identifier for this repository
- `--api-url` (optional): Override API endpoint

**Output:**
```
📂 Indexing repository: /path/to/repo
✅ Indexing Complete!
   Files: 527 | Functions: 4,463 | Classes: 162
   Embeddings: 5,152 | Duration: 399126ms
```

### Query Code (Natural Language)

Search using natural language:

```bash
# Basic query
armyknife code query "authentication patterns"

# Search specific repository
armyknife code query "error handling" --repo-id 14

# Limit results
armyknife code query "database queries" --limit 5
```

### Hybrid Search (Vector + Keyword)

Combine semantic and keyword matching:

```bash
armyknife code hybrid "connection pooling optimization"
```

### View Metrics

Monitor system performance:

```bash
armyknife code metrics
```

Output includes:
- Cache hit rate and performance
- Query latency (p50, p95, p99)
- Index statistics

### Repository Management

```bash
# List indexed repositories
armyknife code repo list

# Get repository info
armyknife code repo info 14

# Delete repository from index
armyknife code repo delete 14
```

## Architecture

```
┌─────────────────────────────────────┐
│  ArmyKnife CLI (You are here)       │
└────────────┬────────────────────────┘
             │
             ▼ HTTP REST API
┌─────────────────────────────────────┐
│  Code Intelligence Backend          │
│  ├─ Parser (tree-sitter)            │
│  ├─ Embeddings (OpenAI/Ollama)      │
│  ├─ Storage (PostgreSQL + pgvector) │
│  └─ Cache (Redis)                   │
└─────────────────────────────────────┘
```

## Documentation

For comprehensive documentation, see:

- **[Complete Guide](https://gist.github.com/armyknife-tools/05e48e844e51db804f788ab220c4b782)** - How to use, what it does, why it exists, and technology details
- **[Deployment Guide](./docs/DEPLOYMENT.md)** - Production setup and scaling
- **[API Reference](./docs/API.md)** - Detailed endpoint documentation
- **[Examples](./examples/)** - Real-world usage examples

## Performance

Query latency targets (achieved):

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| p50 | 100ms | 50ms | ✅ 2x Better |
| p95 | 200ms | 150ms | ✅ 33% Better |
| p99 | 500ms | 300ms | ✅ 40% Better |

Cache performance:
- Cold start: 50-150ms
- Warm cache hit: <10ms
- Target warm hit rate: >75%

## Supported Languages

### Fully Supported (with function/class extraction)
- TypeScript/JavaScript (.ts, .tsx, .js, .jsx)
- Go (.go)
- Python (.py)
- Rust (.rs)
- Java (.java)
- C/C++ (.c, .cpp, .h, .hpp)
- Ruby (.rb)
- PHP (.php)

### Basic Support (text search only)
- JSON (.json)
- YAML (.yaml, .yml)
- Markdown (.md)
- Shell (.sh, .bash)
- SQL (.sql)

## Use Cases

### 1. Onboarding New Developers
```bash
armyknife code query "How do we handle authentication?"
armyknife code query "Error handling patterns"
```

### 2. Code Review Automation
```bash
# Find all database queries to verify consistency
armyknife code query "database transaction handling" --limit 20
```

### 3. Framework Migration
```bash
# Find all Redux implementations before migrating to Context API
armyknife code query "Redux state management"
```

### 4. Code Consistency
```bash
# Find all HTTP error handling
armyknife code hybrid "HTTP error response handling"
```

## Development

### Prerequisites

- Go 1.18 or later
- PostgreSQL (if running backend locally)
- Redis (if running backend locally)

### Building

```bash
# Build binary
go build -o armyknife ./cmd

# Run tests
go test ./...

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o armyknife-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o armyknife-darwin-amd64
GOOS=windows GOARCH=amd64 go build -o armyknife-windows-amd64.exe
```

## Related Projects

ArmyKnife is available in multiple languages for different ecosystems:

| Language | Project | Package | Repo |
|----------|---------|---------|------|
| **Go** | armyknife-cli | [GitHub Releases](https://github.com/armyknifelabs-platform/armyknife-cli/releases) | [armyknife-cli](https://github.com/armyknifelabs-platform/armyknife-cli) |
| **Python** | armyknife-py | [PyPI](https://pypi.org/project/armyknife-cli/) | [armyknife-py](https://github.com/armyknifelabs-platform/armyknife-py) |
| **TypeScript** | armyknife-ts | [NPM](https://www.npmjs.com/package/armyknife-cli) | [armyknife-ts](https://github.com/armyknifelabs-platform/armyknife-ts) |

All clients provide identical functionality with language-native idioms and best practices.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## Roadmap

### Phase 1 (Complete ✅)
- [x] Code indexing and semantic search
- [x] Hybrid search (vector + keyword)
- [x] Performance metrics
- [x] Multi-repository support
- [x] Go CLI (armyknife-cli)
- [x] Python client library (armyknife-py) - [GitHub](https://github.com/armyknifelabs-platform/armyknife-py)
- [x] TypeScript/JavaScript client (armyknife-ts) - [GitHub](https://github.com/armyknifelabs-platform/armyknife-ts)

### Phase 2 (In Progress)
- [ ] Code diff summarization
- [ ] Automatic refactoring suggestions
- [ ] IDE plugins (VSCode, JetBrains)
- [ ] Batch query API for bulk operations

### Phase 3 (Planned)
- [ ] Test generation from code context
- [ ] Documentation generation from codebase
- [ ] Multi-tenant support for SaaS
- [ ] Advanced code metrics and insights
- [ ] CI/CD integration plugins

## License

MIT License - see [LICENSE](./LICENSE) for details

## Support

- **Issues**: [GitHub Issues](https://github.com/armyknifelabs-platform/armyknife-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/armyknifelabs-platform/armyknife-cli/discussions)
- **Documentation**: [Complete Guide](https://gist.github.com/armyknife-tools/05e48e844e51db804f788ab220c4b782)

## About

ArmyKnife is developed by [armyknifelabs](https://github.com/armyknifelabs-platform) as part of the enterprise development platform.

**Current Status:** Production Ready (v1.0.0)

---

**Questions?** Check the [complete guide on GitHub Gist](https://gist.github.com/armyknife-tools/05e48e844e51db804f788ab220c4b782) for comprehensive documentation.
