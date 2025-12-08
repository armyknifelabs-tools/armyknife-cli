# Code Intelligence API - CLI to Endpoint Mapping

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     ARMYKNIFE CLI (The Cockpit)                        │
│                                                                         │
│  armyknife review <command> [options]                                   │
│                                                                         │
│  Commands:                                                              │
│    code      - AI code review                                           │
│    pr        - Pull Request review                                      │
│    security  - Security vulnerability scan                              │
│    patterns  - Pattern detection                                        │
│    standards - Code standards check                                     │
│    architecture - Architecture analysis                                 │
│    flow      - Code flow diagram generation                             │
│    generate-pr - AI-assisted PR creation                                │
│    check-pr  - PR merge readiness check                                 │
│                                                                         │
│  Modes:                                                                 │
│    --local   → Uses local Ollama/node-llm (private, offline)           │
│    --cloud   → Uses API Gateway (powerful models, default)              │
└───────────────────────────────────┬─────────────────────────────────────┘
                                    │
                                    │ HTTPS
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     SEIP PLATFORM API (The Memory)                     │
│                     https://api.codevelocity.dev/api/v1                │
│                                                                         │
│  Base URL: /api/v1/ai/review/*                                         │
│                                                                         │
│  All endpoints store results in PostgreSQL for historical analysis     │
│  RAG embeddings used for semantic code understanding                    │
│  Analysis cached with configurable TTL                                  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## CLI Command → API Endpoint Mapping

### 1. Code Review

**CLI Command:**
```bash
armyknife review code <file-or-directory>
armyknife review code src/services/auth.ts
armyknife review code src/services/ --local --output review.md
```

**API Endpoint:**
```
POST /api/v1/ai/review/code
```

**Request Body:**
```json
{
  "code": "<source code content>",
  "target": "src/services/auth.ts",
  "reviewType": "comprehensive",
  "provider": "gateway",  // or "local" for Ollama
  "model": "claude-3-sonnet",  // optional
  "options": {
    "checkBugs": true,
    "checkStyle": true,
    "checkPerformance": true,
    "checkSecurity": true,
    "suggestRefactors": true
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "summary": "Code review summary...",
    "score": 85,
    "issues": [
      {
        "severity": "high",
        "type": "bug",
        "message": "Potential null pointer exception",
        "line": 42,
        "suggestion": "Add null check before accessing property"
      }
    ],
    "suggestions": [
      "Consider extracting authentication logic to a separate service",
      "Add input validation for user credentials"
    ],
    "metrics": {
      "complexity": "medium",
      "maintainability": "high",
      "testability": "medium"
    }
  }
}
```

---

### 2. Pull Request Review

**CLI Command:**
```bash
armyknife review pr <pr-number> --owner <org> --repo <repo>
armyknife review pr 123 --owner myorg --repo myrepo
armyknife review pr 456 --owner myorg --repo myrepo --local
```

**API Endpoint:**
```
POST /api/v1/ai/review/pr
```

**Request Body:**
```json
{
  "owner": "myorg",
  "repo": "myrepo",
  "prNumber": 123,
  "provider": "gateway",
  "model": "gpt-4",
  "options": {
    "checkCode": true,
    "checkTests": true,
    "checkSecurity": true,
    "checkDocs": true,
    "checkBreakingChanges": true
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "summary": "PR review summary...",
    "verdict": "approve",  // approve | request_changes | reject
    "changesAnalysis": {
      "filesChanged": 5,
      "additions": 150,
      "deletions": 30,
      "impactedAreas": ["authentication", "api"]
    },
    "issues": [...],
    "suggestions": [...],
    "testCoverage": {
      "status": "sufficient",
      "coverage": 82
    },
    "securityImplications": {
      "severity": "low",
      "issues": []
    }
  }
}
```

---

### 3. Security Scan

**CLI Command:**
```bash
armyknife review security <file-or-directory>
armyknife review security src/ --standard owasp-top-10
armyknife review security . --output security-report.md
```

**API Endpoint:**
```
POST /api/v1/ai/review/security
```

**Request Body:**
```json
{
  "code": "<source code content>",
  "target": "src/",
  "standard": "owasp-top-10",
  "provider": "gateway",
  "checks": [
    "injection",
    "xss",
    "authentication",
    "authorization",
    "secrets",
    "cryptography",
    "dependencies"
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "securityScore": 78,
    "standard": "owasp-top-10",
    "vulnerabilities": [
      {
        "type": "SQL Injection",
        "severity": "critical",
        "category": "A03:2021-Injection",
        "description": "User input directly concatenated in SQL query",
        "file": "src/db/queries.ts",
        "line": 45,
        "code": "const query = `SELECT * FROM users WHERE id = ${userId}`",
        "fix": "Use parameterized queries: db.query('SELECT * FROM users WHERE id = $1', [userId])",
        "cweId": "CWE-89"
      }
    ],
    "secretsFound": [
      {
        "type": "API Key",
        "file": "src/config.ts",
        "line": 12,
        "pattern": "sk-..."
      }
    ],
    "recommendations": [
      "Implement Content Security Policy headers",
      "Add rate limiting to authentication endpoints"
    ]
  }
}
```

---

### 4. Pattern Detection

**CLI Command:**
```bash
armyknife review patterns <file-or-directory>
armyknife review patterns src/services/
armyknife review patterns . --format json --output patterns.json
```

**API Endpoint:**
```
POST /api/v1/ai/review/patterns
```

**Request Body:**
```json
{
  "code": "<source code content>",
  "target": "src/services/",
  "provider": "gateway",
  "detect": [
    "design_patterns",
    "anti_patterns",
    "framework_patterns",
    "custom_patterns"
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "designPatterns": [
      {
        "name": "Repository Pattern",
        "location": "src/repositories/UserRepository.ts",
        "confidence": 0.95,
        "description": "Data access abstraction layer"
      },
      {
        "name": "Factory Pattern",
        "location": "src/factories/ServiceFactory.ts",
        "confidence": 0.88
      }
    ],
    "antiPatterns": [
      {
        "name": "God Class",
        "location": "src/services/MainService.ts",
        "severity": "high",
        "metrics": {
          "methods": 45,
          "linesOfCode": 1200,
          "dependencies": 15
        },
        "suggestion": "Split into smaller, focused services (AuthService, UserService, etc.)"
      }
    ],
    "frameworkPatterns": [
      {
        "name": "Express Middleware Chain",
        "location": "src/middleware/",
        "instances": 8
      }
    ],
    "statistics": {
      "patternsFound": 12,
      "antiPatternsFound": 3,
      "codeQuality": "good"
    }
  }
}
```

---

### 5. Standards Check

**CLI Command:**
```bash
armyknife review standards <file-or-directory>
armyknife review standards src/ --standard typescript-strict
armyknife review standards . --output standards-report.md
```

**API Endpoint:**
```
POST /api/v1/ai/review/standards
```

**Request Body:**
```json
{
  "code": "<source code content>",
  "target": "src/",
  "standardSet": "typescript-strict",
  "provider": "gateway",
  "checks": [
    "naming",
    "organization",
    "documentation",
    "error_handling",
    "logging",
    "testing"
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "complianceScore": 72,
    "standardSet": "typescript-strict",
    "violations": [
      {
        "rule": "naming.function",
        "file": "src/utils.ts",
        "line": 15,
        "message": "Function 'getData' should have a more descriptive name",
        "suggestion": "Rename to 'fetchUserProfile' or 'getUserData'"
      },
      {
        "rule": "documentation.public-api",
        "file": "src/services/AuthService.ts",
        "line": 1,
        "message": "Public class missing JSDoc documentation",
        "suggestion": "Add @description and @example tags"
      }
    ],
    "passed": [
      "error_handling.try-catch",
      "logging.structured",
      "organization.file-structure"
    ],
    "recommendations": [
      "Add pre-commit hooks for automated standards checking",
      "Configure ESLint with @typescript-eslint/recommended"
    ]
  }
}
```

---

### 6. Architecture Analysis

**CLI Command:**
```bash
armyknife review architecture <file-or-directory>
armyknife review architecture src/
armyknife review architecture . --format mermaid --output architecture.md
```

**API Endpoint:**
```
POST /api/v1/ai/review/architecture
```

**Request Body:**
```json
{
  "code": "<source code content>",
  "target": "src/",
  "outputFormat": "mermaid",
  "provider": "gateway",
  "analyze": [
    "layers",
    "dependencies",
    "modules",
    "api_design",
    "data_flow"
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "summary": "3-tier architecture with clear separation of concerns...",
    "layers": [
      {
        "name": "Presentation",
        "path": "src/routes/",
        "components": ["api.routes.ts", "auth.routes.ts"]
      },
      {
        "name": "Business Logic",
        "path": "src/services/",
        "components": ["AuthService.ts", "UserService.ts"]
      },
      {
        "name": "Data Access",
        "path": "src/repositories/",
        "components": ["UserRepository.ts"]
      }
    ],
    "dependencies": {
      "internal": {
        "routes": ["services"],
        "services": ["repositories"],
        "repositories": []
      },
      "external": ["express", "pg", "redis"]
    },
    "diagram": "graph TD\n    A[Routes] --> B[Services]\n    B --> C[Repositories]\n    C --> D[(Database)]",
    "suggestions": [
      "Consider adding a caching layer between services and repositories",
      "Extract validation logic into a separate layer"
    ],
    "metrics": {
      "coupling": "low",
      "cohesion": "high",
      "maintainability": "high"
    }
  }
}
```

---

### 7. Code Flow Diagram

**CLI Command:**
```bash
armyknife review flow <file>
armyknife review flow src/main.go --format mermaid
armyknife review flow src/server.ts --output flow-diagram.md
```

**API Endpoint:**
```
POST /api/v1/ai/review/flow
```

**Request Body:**
```json
{
  "code": "<source code content>",
  "target": "src/main.go",
  "outputFormat": "mermaid",
  "provider": "gateway",
  "analyze": [
    "entry_points",
    "exit_points",
    "control_flow",
    "call_graph",
    "data_flow"
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "entryPoints": [
      {
        "name": "main()",
        "type": "function",
        "file": "src/main.go",
        "line": 15
      },
      {
        "name": "handleRequest",
        "type": "http_handler",
        "file": "src/handlers.go",
        "line": 42
      }
    ],
    "exitPoints": [
      {
        "name": "os.Exit(0)",
        "type": "exit",
        "file": "src/main.go",
        "line": 100
      },
      {
        "name": "return response",
        "type": "return",
        "file": "src/handlers.go",
        "line": 78
      }
    ],
    "callGraph": {
      "main": ["initDB", "initServer", "startServer"],
      "startServer": ["handleRequest", "handleHealth"],
      "handleRequest": ["validateInput", "processRequest", "sendResponse"]
    },
    "flowDiagram": "flowchart TD\n    A[main] --> B[initDB]\n    A --> C[initServer]\n    C --> D[startServer]\n    D --> E{handleRequest}\n    E --> F[validateInput]\n    F --> G[processRequest]\n    G --> H[sendResponse]",
    "dataFlow": [
      {
        "variable": "config",
        "from": "main",
        "to": ["initDB", "initServer"],
        "type": "Config"
      }
    ]
  }
}
```

---

### 8. AI-Assisted PR Generation

**CLI Command:**
```bash
armyknife review generate-pr --title "Add authentication"
armyknife review generate-pr --branch feature/auth --base main
armyknife review generate-pr --analyze-changes --draft
```

**API Endpoint:**
```
POST /api/v1/ai/review/generate-pr
```

**Request Body:**
```json
{
  "title": "Add authentication feature",
  "branch": "feature/auth",
  "base": "main",
  "analyzeChanges": true,
  "draft": false,
  "provider": "gateway",
  "options": {
    "generateDescription": true,
    "generateTestPlan": true,
    "suggestReviewers": true,
    "linkIssues": true
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "title": "feat(auth): Add JWT-based authentication system",
    "description": "## Summary\n\nThis PR implements a complete JWT-based authentication system including:\n- User registration and login endpoints\n- Token generation and validation\n- Middleware for protected routes\n\n## Changes\n- Added `AuthService` for authentication logic\n- Added `JWTMiddleware` for route protection\n- Added user model and migrations\n\n## Testing\n- Unit tests for AuthService\n- Integration tests for auth endpoints",
    "testPlan": "## Test Plan\n\n- [ ] Test user registration with valid data\n- [ ] Test user registration with invalid data\n- [ ] Test login with correct credentials\n- [ ] Test login with incorrect credentials\n- [ ] Test protected route with valid token\n- [ ] Test protected route without token",
    "suggestedReviewers": ["@senior-dev", "@security-lead"],
    "linkedIssues": ["#42 - Implement user authentication"],
    "prUrl": "https://github.com/myorg/myrepo/pull/123"
  }
}
```

---

### 9. PR Merge Readiness Check

**CLI Command:**
```bash
armyknife review check-pr <pr-number> --owner <org> --repo <repo>
armyknife review check-pr 123 --owner myorg --repo myrepo
armyknife review check-pr 456 --require-tests --require-docs
```

**API Endpoint:**
```
POST /api/v1/ai/review/check-pr
```

**Request Body:**
```json
{
  "owner": "myorg",
  "repo": "myrepo",
  "prNumber": 123,
  "checks": [
    "code_quality",
    "test_coverage",
    "security",
    "breaking_changes",
    "documentation",
    "ci_status"
  ]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "mergeReady": false,
    "readinessScore": 65,
    "blockers": [
      "Test coverage below threshold (72% < 80%)",
      "CI pipeline has failing tests"
    ],
    "warnings": [
      "No documentation updates detected",
      "2 unresolved review comments"
    ],
    "checks": {
      "code_quality": { "status": "pass", "score": 85 },
      "test_coverage": { "status": "fail", "value": 72, "required": 80 },
      "security": { "status": "pass", "vulnerabilities": 0 },
      "breaking_changes": { "status": "warn", "detected": true },
      "documentation": { "status": "warn", "updated": false },
      "ci_status": { "status": "fail", "failures": ["unit-tests"] }
    },
    "recommendation": "Fix failing tests and increase coverage before merging"
  }
}
```

---

## Backend Endpoints to Implement

The following endpoints need to be implemented in the backend (`armyknifelabs-platform-backend`):

| Endpoint | Method | File to Create |
|----------|--------|----------------|
| `/api/v1/ai/review/code` | POST | `src/routes/ai/review.routes.ts` |
| `/api/v1/ai/review/pr` | POST | `src/routes/ai/review.routes.ts` |
| `/api/v1/ai/review/security` | POST | `src/routes/ai/review.routes.ts` |
| `/api/v1/ai/review/patterns` | POST | `src/routes/ai/review.routes.ts` |
| `/api/v1/ai/review/standards` | POST | `src/routes/ai/review.routes.ts` |
| `/api/v1/ai/review/architecture` | POST | `src/routes/ai/review.routes.ts` |
| `/api/v1/ai/review/flow` | POST | `src/routes/ai/review.routes.ts` |
| `/api/v1/ai/review/generate-pr` | POST | `src/routes/ai/review.routes.ts` |
| `/api/v1/ai/review/check-pr` | POST | `src/routes/ai/review.routes.ts` |

## Service Classes to Create

```
src/services/
├── CodeReviewService.ts      # Code review logic + LLM integration
├── SecurityScanService.ts    # OWASP scanning + vulnerability detection
├── PatternDetectionService.ts # Design pattern recognition
├── ArchitectureAnalyzer.ts   # Architecture analysis + diagram generation
├── CodeFlowAnalyzer.ts       # Control flow + call graph generation
└── PRService.ts              # PR generation and validation
```

## LLM Provider Integration

All review services support dual-mode operation:

```typescript
interface ReviewOptions {
  provider: 'gateway' | 'local';  // Cloud vs Local AI
  model?: string;                  // Specific model override
}

// Gateway mode → Uses LLM Gateway (Claude, GPT-4, etc.)
// Local mode → Uses Ollama cluster or node-llm
```

## Caching Strategy

All analysis results are cached in PostgreSQL:

```sql
CREATE TABLE ai_review_cache (
  id SERIAL PRIMARY KEY,
  content_hash VARCHAR(64) NOT NULL,
  review_type VARCHAR(50) NOT NULL,
  result JSONB NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  expires_at TIMESTAMP,
  UNIQUE(content_hash, review_type)
);
```

Default TTL: 24 hours (configurable per review type)

---

## Usage Summary

```bash
# The Cockpit - Command Center
armyknife review code src/           # Review code quality
armyknife review pr 123              # Review pull request
armyknife review security src/       # OWASP security scan
armyknife review patterns src/       # Detect design patterns
armyknife review standards src/      # Check code standards
armyknife review architecture src/   # Analyze architecture
armyknife review flow src/main.go    # Generate flow diagram
armyknife review generate-pr         # AI-assisted PR creation
armyknife review check-pr 123        # PR merge readiness

# Use local AI for private/offline analysis
armyknife review code src/ --local

# Output to file
armyknife review security src/ --output security-report.md
```
