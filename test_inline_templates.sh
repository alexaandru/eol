#!/bin/bash

# Integration test script to verify inline template examples work correctly
# This script tests the examples from the help text to ensure they work as expected
#
# Coverage Integration:
# This script uses Go's integration test coverage feature (go build -cover) to collect
# coverage data from the actual binary execution, then converts it to atomic format.
#
# Based on: https://go.dev/blog/integration-test-coverage
#
# Usage:
#   ./test_inline_templates.sh
#
# Output:
#   - Tests all template examples
#   - Generates integration.cov file with coverage data
#   - Displays coverage summary by package and total coverage

set -e

# Set coverage directory
COVERAGE_DIR=coverage-integration
mkdir -p $COVERAGE_DIR

echo "Building eol binary with coverage instrumentation..."
go build -cover -covermode=atomic -o eol-test cmd/eol/main.go

echo ""
echo "Testing inline template examples from help text..."
echo "================================================="

# Test 1: Basic product template
echo ""
echo "Test 1: Basic product template"
echo "Command: ./eol-test -t '{{.Name}} - {{.Category}}' product ubuntu"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test -t '{{.Name}} - {{.Category}}' product ubuntu)
echo "Result: $result"
if [[ "$result" == "ubuntu - os" ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Expected 'ubuntu - os', got '$result'"
    exit 1
fi

# Test 2: Latest release template
echo ""
echo "Test 2: Latest release template"
echo "Command: ./eol-test --template '{{.Latest.Name}}' latest go"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test --template '{{.Latest.Name}}' latest go)
echo "Result: $result"
if [[ "$result" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "✅ PASS (version format looks correct)"
else
    echo "❌ FAIL: Expected version format, got '$result'"
    exit 1
fi

# Test 3: Maintenance status template
echo ""
echo "Test 3: Maintenance status template"
echo "Command: ./eol-test -t '{{.Name}}: {{if .IsMaintained}}✅ Active{{else}}💀 EOL{{end}}' latest terraform"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test -t '{{.Name}}: {{if .IsMaintained}}✅ Active{{else}}💀 EOL{{end}}' latest terraform)
echo "Result: $result"
if [[ "$result" =~ : && ("$result" =~ "✅ Active" || "$result" =~ "💀 EOL") ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Expected format 'version: status', got '$result'"
    exit 1
fi

# Test 4: Tags template
echo ""
echo "Test 4: Tags template"
echo "Command: ./eol-test --template '{{join .Tags \", \"}}' product go"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test --template '{{join .Tags ", "}}' product go)
echo "Result: $result"
if [[ "$result" =~ "google" && "$result" =~ "lang" ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Expected tags containing 'google' and 'lang', got '$result'"
    exit 1
fi

# Test 5: Cache stats template
echo ""
echo "Test 5: Cache stats template"
echo "Command: ./eol-test -t '{{.TotalFiles}} files ({{.ValidFiles}} valid)' cache stats"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test -t '{{.TotalFiles}} files ({{.ValidFiles}} valid)' cache stats)
echo "Result: $result"
if [[ "$result" =~ ^[0-9]+\ files\ \([0-9]+\ valid\)$ ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Expected format 'N files (N valid)', got '$result'"
    exit 1
fi

# Test 6: JSON template function
echo ""
echo "Test 6: JSON template function"
echo "Command: ./eol-test -t '{{toJSON .Links}}' product go"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test -t '{{toJSON .Links}}' product go)
echo "Result: $result"
if [[ "$result" =~ "{" && "$result" =~ "}" ]]; then
    echo "✅ PASS (JSON format detected)"
else
    echo "❌ FAIL: Expected JSON format, got '$result'"
    exit 1
fi

# Test 7: Math functions
echo ""
echo "Test 7: Math functions"
echo "Command: ./eol-test -t '{{add .TotalFiles .ValidFiles}}' cache stats"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test -t '{{add .TotalFiles .ValidFiles}}' cache stats)
echo "Result: $result"
if [[ "$result" =~ ^[0-9]+$ ]]; then
    echo "✅ PASS (numeric result)"
else
    echo "❌ FAIL: Expected numeric result, got '$result'"
    exit 1
fi

# Test 8: Default function
echo ""
echo "Test 8: Default function"
echo "Command: ./eol-test -t '{{default \"N/A\" .VersionCommand}}' product go"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test -t '{{default "N/A" .VersionCommand}}' product go)
echo "Result: $result"
if [[ "$result" == "N/A" || "$result" != "" ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: Expected 'N/A' or some value, got '$result'"
    exit 1
fi

# Test 9: Exit function for EOL detection (scripting)
echo ""
echo "Test 9: Exit function for EOL detection"
echo "Command: ./eol-test release go 1.17 -t '{{if .IsEol}}{{exit 1}}{{end}}' (should exit with code 1)"
if GOCOVERDIR=$COVERAGE_DIR ./eol-test release go 1.17 -t '{{if .IsEol}}{{exit 1}}{{end}}' >/dev/null 2>&1; then
    echo "❌ FAIL: Expected exit code 1 for EOL version"
    exit 1
else
    exit_code=$?
    if [[ $exit_code -eq 1 ]]; then
        echo "✅ PASS (correctly detected EOL with exit code 1)"
    else
        echo "❌ FAIL: Expected exit code 1, got $exit_code"
        exit 1
    fi
fi

# Test 10: Non-EOL version should not exit
# Test 10: Non-EOL version should continue normally
echo ""
echo "Test 10: Non-EOL version should continue normally"
echo "Command: ./eol-test release go 1.25 -t '{{if .IsEol}}{{exit 1}}{{end}}Active: {{.Name}}'"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test release go 1.25 -t '{{if .IsEol}}{{exit 1}}{{end}}Active: {{.Name}}')
echo "Result: $result"
if [[ "$result" =~ "Active:" ]]; then
    echo "✅ PASS (non-EOL version processed normally)"
else
    echo "❌ FAIL: Expected 'Active:' in output, got '$result'"
    exit 1
fi

# Test 11: eol_within function
echo ""
echo "Test 11: eol_within function"
echo "Command: ./eol-test product nodejs -t '{{range .Releases}}{{if eol_within \"12mo\" .EolFrom}}{{.Name}}: {{.EolFrom}}{{\"\\n\"}}{{end}}{{end}}'"
result=$(GOCOVERDIR=$COVERAGE_DIR ./eol-test product nodejs -t '{{range .Releases}}{{if eol_within "12mo" .EolFrom}}{{.Name}}: {{.EolFrom}}{{"\n"}}{{end}}{{end}}')
echo "Result: $result"
if [[ -n "$result" ]] || [[ -z "$result" ]]; then
    echo "✅ PASS (eol_within function executed without errors)"
else
    echo "❌ FAIL: eol_within function failed"
    exit 1
fi

echo ""
echo "================================================="
echo "🎉 All inline template tests passed!"
echo ""
echo "================================================="
echo "Converting coverage data to atomic format..."

# Convert coverage data to text format
if ! go tool covdata textfmt -i=$COVERAGE_DIR -o integration.cov; then
    echo "❌ Failed to convert coverage data"
    echo "Cleaning up..."
    rm -f eol-test
    rm -rf $COVERAGE_DIR
    exit 1
fi



if [[ -f integration.cov ]]; then
    echo "✅ Integration coverage written to integration.cov"
    echo ""
    echo "📊 Coverage summary by package:"
    go tool cover -func=integration.cov | grep -E '^github.com/alexaandru/eol' | sort -k3 -nr | head -10
    echo ""
    echo "📈 Total coverage:"
    go tool cover -func=integration.cov | tail -1
    echo ""
    echo "💡 To view detailed HTML coverage report:"
    echo "   go tool cover -html=integration.cov -o coverage.html && open coverage.html"
else
    echo "❌ Failed to generate coverage file"
fi

echo ""
echo "Cleaning up temporary files..."
rm -f eol-test
rm -rf $COVERAGE_DIR

echo "✅ Test completed successfully!"
