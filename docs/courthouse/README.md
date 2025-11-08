# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2025 Controle Digital Ltda

# DictaMesh Courthouse - Code Compliance System

The DictaMesh Courthouse is an automated compliance review system that enforces code quality policies through AI agent collaboration (A2A - Agent-to-Agent protocols).

## Overview

The Courthouse system consists of:

- **Judge Agents**: AI agents that review code for compliance with framework policies
- **A2A Protocols**: Standardized communication protocols between agents
- **Enforcement Tools**: Pre-commit hooks, CI/CD integrations, and IDE extensions
- **Exception Management**: Process for handling temporary policy exceptions

## Purpose

Ensure all DictaMesh framework code adheres to:

1. **File Size Limits**: Max 500 lines, preferred 350 lines
2. **Code Reusability**: DRY principles, no duplication
3. **Dependency Management**: Use of appropriate external libraries
4. **Code Organization**: Clear structure and naming conventions
5. **Function Size**: Max 50 lines per function

See [AGENT.md](../../AGENT.md) for complete policy details.

## Directory Structure

```
docs/courthouse/
├── README.md                           # This file
├── agents/                            # Judge agent definitions
│   ├── code-compliance-judge.md       # Main compliance judge (A2A protocol)
│   └── [future-agents].md             # Additional specialized judges
├── templates/                         # Templates for compliance workflow
│   ├── exception-request.yml          # Request exception to policy
│   └── compliance-report.md           # Report template
├── guides/                            # Implementation guides
│   ├── integration-guide.md           # How to integrate compliance checks
│   ├── code-splitting.md              # Patterns for splitting large files
│   └── refactoring-patterns.md        # Best practices for refactoring
└── examples/                          # Example compliance scenarios
    ├── pre-commit-hook.sh             # Git hook example
    └── github-action.yml              # CI/CD example
```

## Quick Start

### For Developers

**Before committing code:**

```bash
# Check compliance of specific file
@compliance-judge review pkg/services/user_service.go

# Check all staged files
@compliance-judge review --scope=staged

# Run strict check (no warnings allowed)
@compliance-judge review --scope=staged --strict
```

**File size check:**

```bash
# Quick check: count code lines (excludes headers, imports, blanks)
wc -l pkg/services/user_service.go
# If over 350 lines, consider refactoring
# If over 500 lines, MUST refactor before commit
```

### For Repository Admins

**Install pre-commit hook:**

```bash
cp docs/courthouse/examples/pre-commit-hook.sh .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

**Add CI/CD check:**

See \`docs/courthouse/examples/github-action.yml\`

## Judge Agents

### Code Compliance Judge

**Agent ID**: \`compliance-judge-001\`
**Status**: Active
**Version**: 1.0.0

**Responsibilities**:
- Enforce file size limits
- Detect code duplication
- Validate dependency usage
- Check code organization
- Monitor function complexity

**Full Protocol**: [agents/code-compliance-judge.md](agents/code-compliance-judge.md)

### Future Judges (Planned)

- **Security Compliance Judge**: Check for security vulnerabilities
- **Performance Compliance Judge**: Identify performance anti-patterns
- **API Compliance Judge**: Validate API design standards
- **Documentation Compliance Judge**: Ensure adequate documentation

## A2A Protocol

Agents communicate using structured JSON protocol:

```json
{
  "protocol": "A2A-Compliance-Review",
  "version": "1.0",
  "request_id": "unique-id",
  "requesting_agent": "agent-identifier",
  "review_type": "pre-commit|pre-pr|full-audit",
  "scope": {
    "files": ["file1.go", "file2.js"]
  }
}
```

See individual agent documentation for complete protocol specifications.

## Exception Process

If you need to temporarily violate a policy:

1. **Create exception request**: Use template from \`templates/exception-request.yml\`
2. **Justify necessity**: Explain why exception is needed
3. **Provide refactoring plan**: Timeline to resolve issue
4. **Get approval**: Tech lead review required
5. **Set expiration**: Max 2 sprint cycles
6. **Track progress**: Link to refactoring tickets

**Exception file location**: \`.compliance-exceptions/\`

## Compliance Metrics

The Courthouse system tracks:

- **Compliance Score**: Overall project health (0-100)
- **Violation Trends**: Improving/degrading over time
- **File Size Distribution**: How many files near limits
- **Code Duplication**: Percentage of duplicated code
- **Dependency Health**: Outdated/vulnerable packages

## Severity Levels

| Level | Status | Blocking | Action |
|-------|--------|----------|--------|
| 0 | PASS | No | None |
| 1 | WARNING | No | Fix in next sprint |
| 2 | VIOLATION | No* | Mandatory fix |
| 3 | CRITICAL | Yes | Immediate fix required |

*Blocks new features but not current PR if documented

## Resources

- [AGENT.md](../../AGENT.md) - Full policy specification
- [Code Compliance Judge Protocol](agents/code-compliance-judge.md)

---

**System Status**: Active
**Maintained By**: DictaMesh Architecture Team
**Last Updated**: 2025-11-08
