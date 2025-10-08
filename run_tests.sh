#!/bin/bash

# CSV Email Validator API - Test Runner
# This script runs all unit tests and provides coverage information

echo "🧪 Running CSV Email Validator API Tests"
echo "========================================"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed or not in PATH"
    exit 1
fi

# Clean up any previous test artifacts
echo "🧹 Cleaning up previous test artifacts..."
go clean -testcache

# Run tests with verbose output and coverage
echo "🚀 Running all tests with coverage..."
go test -v -coverprofile=coverage.out ./...

# Check if tests passed
if [ $? -eq 0 ]; then
    echo "✅ All tests passed!"
    
    # Generate coverage report
    echo "📊 Generating coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    
    # Show coverage summary
    echo "📈 Coverage Summary:"
    go tool cover -func=coverage.out | tail -1
    
    echo ""
    echo "📁 Coverage report saved to: coverage.html"
    echo "🎉 Test run completed successfully!"
else
    echo "❌ Some tests failed!"
    exit 1
fi
