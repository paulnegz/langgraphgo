#!/bin/bash

# CI-matching lint script
# This script replicates the exact golangci-lint configuration used in CI
# to catch all errors locally before pushing

set -e

echo "🔍 Running golangci-lint with CI configuration..."
echo "📋 Using .golangci.yaml config"
echo "⏱️  Timeout: 5m (same as CI)"

# Install/update to latest version (matching CI)
echo "🔄 Ensuring latest golangci-lint version..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run with exact CI parameters
echo "🚀 Running linter..."
~/go/bin/golangci-lint run --timeout=5m --config=.golangci.yaml ./...

if [ $? -eq 0 ]; then
    echo "✅ All lint checks passed! Safe to push."
else
    echo "❌ Lint checks failed. Fix errors before pushing."
    exit 1
fi