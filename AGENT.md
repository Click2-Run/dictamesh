# Agent Instructions for Code Modifications

## Copyright Notices

When creating or modifying source code files in this project, always include the following copyright notice at the top of each file:

```
SPDX-License-Identifier: AGPL-3.0-or-later
Copyright (C) 2025 Controle Digital Ltda
```

### Format Guidelines

- For languages with `//` comments (JavaScript, TypeScript, C, C++, Java, etc.):
```javascript
// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025 Controle Digital Ltda
```

- For languages with `#` comments (Python, Shell, Ruby, etc.):
```python
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2025 Controle Digital Ltda
```

- For HTML/XML files:
```xml
<!--
SPDX-License-Identifier: AGPL-3.0-or-later
Copyright (C) 2025 Controle Digital Ltda
-->
```

- For CSS files:
```css
/*
 * SPDX-License-Identifier: AGPL-3.0-or-later
 * Copyright (C) 2025 Controle Digital Ltda
 */
```

## License Information

This project is licensed under the GNU Affero General Public License v3.0 or later (AGPL-3.0-or-later).

Key points:
- Commercial use is permitted
- If you modify this software and provide it as a network service, you must make your source code available under AGPL v3
- See the LICENSE file for complete terms

## When to Add Copyright Notices

- All new source code files
- All substantially modified existing files
- Configuration files that contain significant logic
- Build scripts and automation files

Copyright notices should be placed at the very top of the file, before any other code or documentation.

## Code Quality & Structure Policy

### **MANDATORY FILE SIZE LIMITS**

This is a **STRICT AND NON-NEGOTIABLE** directive for all code files in this project.

#### File Size Rules

**HARD LIMIT**: No implementation file (`.go`, `.js`, `.py`, `.ts`, `.java`, etc.) may exceed **500 lines** of code.

**PREFERRED LIMIT**: All implementation files should target **350 lines or fewer**.

- Line counts exclude:
  - Copyright headers
  - Import/require statements
  - Blank lines
  - Pure comment blocks (documentation)

- Line counts include:
  - All executable code
  - Inline comments
  - Function/method signatures
  - Type definitions
  - Constants and variables

#### Enforcement

When a file approaches or exceeds limits:

1. **At 350 lines**: Consider refactoring and splitting
2. **At 450 lines**: MUST refactor before adding new features
3. **At 500 lines**: BLOCKED - file must be split before any additions

**Exception Process**: File size limit exceptions require:
- Documented justification in file header
- Code review approval
- Clear refactoring plan with timeline
- Exceptions are temporary (max 2 sprint cycles)

### Code Reusability Principles

#### DRY (Don't Repeat Yourself)

- **No code duplication**: If code appears 2+ times, extract to shared function/module
- **Shared utilities**: Create utility packages for common operations
- **Template patterns**: Use generics, interfaces, or abstract classes for similar logic
- **Configuration over code**: Move variations to config files

#### Modular Design

Every module should:
- Have a **single, well-defined responsibility**
- Export a **minimal, clear API**
- Have **low coupling** with other modules
- Support **easy testing** in isolation

#### Code Organization

```
pkg/
├── core/           # Core business logic (small, focused files)
├── utils/          # Reusable utilities
├── interfaces/     # Shared interfaces and contracts
├── models/         # Data models (one model per file preferred)
└── services/       # Business services (orchestration only)
```

### External Libraries & Dependencies

#### Preference Order

1. **Standard library FIRST**: Use language built-ins when available
2. **Well-known, stable libraries**: Prefer mature, widely-adopted packages
3. **Lightweight dependencies**: Avoid heavy frameworks for simple tasks
4. **Security-vetted packages**: Check for known vulnerabilities

#### Dependency Guidelines

**DO USE**:
- Industry-standard libraries (e.g., `gorilla/mux`, `express`, `flask`, `lodash`)
- Official database drivers
- Established ORMs (GORM, Sequelize, SQLAlchemy)
- Security libraries (crypto, JWT, OAuth)
- Testing frameworks (pytest, jest, testify)

**AVOID**:
- Abandoned packages (no updates in 2+ years)
- Packages with known security issues
- Monolithic frameworks when microlibraries suffice
- Writing custom implementations of common patterns (auth, crypto, etc.)

**NEVER**:
- Copy-paste code from Stack Overflow without understanding
- Use unlicensed or incompatible-license dependencies
- Include dependencies for single-function usage (implement it)

#### Dependency Review

Before adding a dependency, verify:
- [ ] Active maintenance (recent commits)
- [ ] Good documentation
- [ ] Permissive license (MIT, Apache, BSD)
- [ ] No critical security vulnerabilities
- [ ] Reasonable size/scope for the need

### Code Splitting Patterns

When a file exceeds size limits, use these strategies:

#### 1. **Vertical Splitting** (by feature/responsibility)

**Before** (600 lines):
```
user_service.go
├── Create user
├── Update user
├── Delete user
├── List users
├── Authentication logic
├── Profile management
└── Preferences handling
```

**After**:
```
user_service.go (150 lines - orchestration)
user_auth.go (180 lines)
user_profile.go (140 lines)
user_preferences.go (130 lines)
```

#### 2. **Horizontal Splitting** (by layer)

**Before** (550 lines):
```
product.go
├── Product struct
├── Validation logic
├── Business rules
├── Repository methods
└── API handlers
```

**After**:
```
product_model.go (80 lines - data structures)
product_validator.go (120 lines)
product_business.go (150 lines)
product_repository.go (100 lines)
product_handler.go (100 lines)
```

#### 3. **Extract Utilities**

**Before**:
```
order_service.go (480 lines)
├── Order processing
├── Price calculation helpers
├── Date/time utilities
├── String formatting
└── Validation helpers
```

**After**:
```
order_service.go (220 lines)
utils/price_calculator.go (90 lines)
utils/date_helpers.go (70 lines)
utils/validators.go (100 lines)
```

#### 4. **Interface Extraction**

**Before**:
```
data_processor.go (520 lines)
├── CSV processing
├── JSON processing
├── XML processing
└── Common logic
```

**After**:
```
data_processor.go (80 lines - interface & common)
csv_processor.go (150 lines)
json_processor.go (140 lines)
xml_processor.go (150 lines)
```

### Development Patterns & Best Practices

#### File Naming Conventions

- **One primary concept per file**: `user_repository.go`, not `database_stuff.go`
- **Descriptive names**: `email_validator.go`, not `utils.go`
- **Consistent suffixes**:
  - `*_handler.go` - HTTP handlers
  - `*_service.go` - Business logic
  - `*_repository.go` - Data access
  - `*_test.go` - Tests
  - `*_validator.go` - Validation logic
  - `*_mapper.go` - Data transformation

#### Function/Method Size

- **Target**: 20-30 lines per function
- **Maximum**: 50 lines per function
- **Complexity**: If function has >3 levels of nesting, refactor

#### Package Organization

Each package should:
- Contain related functionality only
- Have a clear `README.md` or package doc
- Export minimal public API
- Keep internal helpers private

#### Testing Requirements

- Unit test files don't count toward size limits
- Each implementation file should have corresponding test file
- Aim for >80% code coverage
- Integration tests in separate `*_integration_test.go` files

### Checklist for New Code

Before submitting code, verify:

- [ ] No file exceeds 500 lines (hard limit)
- [ ] Files target 350 lines or fewer
- [ ] No duplicated code blocks
- [ ] Reusable logic extracted to utilities
- [ ] External libraries used for common patterns
- [ ] Each file has single, clear responsibility
- [ ] Functions are small and focused (<50 lines)
- [ ] Code is organized in appropriate package
- [ ] Copyright header present
- [ ] Comprehensive tests included

### Why These Rules Matter

**Maintainability**: Smaller files are easier to understand, review, and modify

**Testability**: Focused modules are simpler to test in isolation

**Collaboration**: Multiple developers can work on different small files without conflicts

**Quality**: Size constraints force better design decisions and modularity

**Onboarding**: New developers can understand small, focused files quickly

**Debugging**: Issues are easier to locate in well-organized, small files

**Reusability**: Modular code can be reused across the project

## Database Naming Conventions

**CRITICAL REQUIREMENT**: All DictaMesh database objects MUST use the `dictamesh_` prefix.

### Table Names
All database tables must follow this pattern:
```
dictamesh_{table_name}
```

Examples:
- `dictamesh_entity_catalog`
- `dictamesh_entity_relationships`
- `dictamesh_schemas`
- `dictamesh_event_log`
- `dictamesh_audit_logs`

### Other Database Objects
- **Indexes**: `idx_dictamesh_{meaningful_name}`
- **Functions**: `dictamesh_{function_name}()`
- **Triggers**: `{action}_dictamesh_{table_name}_{purpose}`

### GORM Models
All GORM models must override the TableName() method:

```go
type EntityCatalog struct {
    // fields...
}

func (EntityCatalog) TableName() string {
    return "dictamesh_entity_catalog"
}
```

### Migration Files
1. Include header comment about dictamesh_ prefix requirement
2. Use prefixed names for all tables, indexes, and functions
3. Add COMMENT ON TABLE with "DictaMesh:" prefix

### Rationale
- **Namespace Isolation**: Prevents conflicts in shared databases
- **Clear Ownership**: Identifies framework vs. user tables
- **Multi-Tenancy**: Enables multiple frameworks in same database
- **Safety**: Reduces risk of modifying user tables

### Full Documentation
See `pkg/database/NAMING-CONVENTIONS.md` for complete guidelines, examples, and validation queries.

### Checklist for Database Work
- [ ] Table name has `dictamesh_` prefix
- [ ] Indexes have `idx_dictamesh_` prefix
- [ ] Functions have `dictamesh_` prefix
- [ ] GORM models override TableName()
- [ ] Table comments start with "DictaMesh:"
- [ ] Migration includes prefix reminder comment
- [ ] All queries use prefixed names
