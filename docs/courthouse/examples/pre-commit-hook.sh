#!/bin/bash
# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2025 Controle Digital Ltda

# DictaMesh Code Compliance Pre-Commit Hook
#
# Installation:
#   cp docs/courthouse/examples/pre-commit-hook.sh .git/hooks/pre-commit
#   chmod +x .git/hooks/pre-commit
#
# This hook enforces DictaMesh code quality policies before commits.

set -e

echo "🏛️  DictaMesh Compliance Judge - Pre-Commit Review"
echo "=================================================="

# Configuration
MAX_LINES=500
PREFERRED_LINES=350
EXCLUDE_PATTERNS=".*_test\.(go|js|py|ts)$|.*\.pb\.go$|.*_generated\.(go|js|py|ts)$"

# Get staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(go|js|py|ts|java)$' || true)

if [ -z "$STAGED_FILES" ]; then
  echo "✅ No code files to review"
  exit 0
fi

echo ""
echo "📋 Files to review:"
echo "$STAGED_FILES" | sed 's/^/  - /'
echo ""

# Counters
VIOLATIONS=0
WARNINGS=0
TOTAL_FILES=0

# Function to count code lines (exclude headers, imports, blank lines)
count_code_lines() {
  local file=$1
  local ext="${file##*.}"
  
  case $ext in
    go)
      # Exclude copyright, package, imports, blank lines
      grep -v '^[[:space:]]*$' "$file" | \
      grep -v '^[[:space:]]*//' | \
      grep -v '^package ' | \
      awk '/^import \(/,/^\)/ {next} /^import / {next} {print}' | \
      wc -l
      ;;
    js|ts)
      # Exclude copyright, imports, blank lines
      grep -v '^[[:space:]]*$' "$file" | \
      grep -v '^[[:space:]]*//' | \
      grep -v '^import ' | \
      grep -v '^export ' | \
      wc -l
      ;;
    py)
      # Exclude copyright, imports, blank lines
      grep -v '^[[:space:]]*$' "$file" | \
      grep -v '^[[:space:]]*#' | \
      grep -v '^import ' | \
      grep -v '^from .* import' | \
      wc -l
      ;;
    *)
      wc -l < "$file"
      ;;
  esac
}

# Check each file
echo "Checking file sizes..."
echo ""

for file in $STAGED_FILES; do
  # Skip excluded patterns
  if echo "$file" | grep -qE "$EXCLUDE_PATTERNS"; then
    echo "  ⏭️  Skipping: $file (excluded pattern)"
    continue
  fi
  
  # Check if file exists (not deleted)
  if [ ! -f "$file" ]; then
    continue
  fi
  
  TOTAL_FILES=$((TOTAL_FILES + 1))
  
  # Count lines
  LINES=$(count_code_lines "$file")
  
  # Evaluate compliance
  if [ "$LINES" -gt "$MAX_LINES" ]; then
    echo "  ❌ VIOLATION: $file"
    echo "     Lines: $LINES (limit: $MAX_LINES)"
    echo "     Action: MUST split file before commit"
    echo ""
    VIOLATIONS=$((VIOLATIONS + 1))
  elif [ "$LINES" -gt "$PREFERRED_LINES" ]; then
    echo "  ⚠️  WARNING: $file"
    echo "     Lines: $LINES (preferred: $PREFERRED_LINES)"
    echo "     Action: Consider refactoring"
    echo ""
    WARNINGS=$((WARNINGS + 1))
  else
    echo "  ✅ PASS: $file ($LINES lines)"
  fi
done

echo ""
echo "=================================================="
echo "Summary:"
echo "  Files reviewed: $TOTAL_FILES"
echo "  Violations: $VIOLATIONS"
echo "  Warnings: $WARNINGS"
echo ""

# Decide outcome
if [ "$VIOLATIONS" -gt 0 ]; then
  echo "❌ COMMIT BLOCKED"
  echo ""
  echo "Critical policy violations detected. Please fix before committing:"
  echo ""
  echo "Options:"
  echo "  1. Split large files using patterns in docs/courthouse/guides/"
  echo "  2. Request exception using docs/courthouse/templates/exception-request.yml"
  echo "  3. Remove large files from this commit (git reset HEAD <file>)"
  echo ""
  echo "For help: see AGENT.md section 'Code Quality & Structure Policy'"
  echo ""
  exit 1
fi

if [ "$WARNINGS" -gt 0 ]; then
  echo "⚠️  WARNINGS DETECTED"
  echo ""
  echo "Files approaching size limits. Consider refactoring soon."
  echo ""
  
  # Optional: prompt user
  read -p "Proceed with commit? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Commit aborted by user"
    exit 1
  fi
fi

echo "✅ Compliance check passed - proceeding with commit"
echo ""
exit 0
