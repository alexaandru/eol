#!/bin/bash

# Test script to verify inline template examples work correctly
# This script tests the examples from the help text to ensure they work as expected

set -e

echo "Building eol binary..."
go build -o eol-test cmd/eol/main.go

echo ""
echo "Testing inline template examples from help text..."
echo "================================================="

# Test 1: Basic product template
echo ""
echo "Test 1: Basic product template"
echo "Command: ./eol-test -t '{{.Name}} - {{.Category}}' product ubuntu"
result=$(./eol-test -t '{{.Name}} - {{.Category}}' product ubuntu)
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
result=$(./eol-test --template '{{.Latest.Name}}' latest go)
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
result=$(./eol-test -t '{{.Name}}: {{if .IsMaintained}}✅ Active{{else}}💀 EOL{{end}}' latest terraform)
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
result=$(./eol-test --template '{{join .Tags ", "}}' product go)
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
result=$(./eol-test -t '{{.TotalFiles}} files ({{.ValidFiles}} valid)' cache stats)
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
result=$(./eol-test -t '{{toJSON .Links}}' product go)
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
result=$(./eol-test -t '{{add .TotalFiles .ValidFiles}}' cache stats)
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
result=$(./eol-test -t '{{default "N/A" .VersionCommand}}' product go)
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
if ./eol-test release go 1.17 -t '{{if .IsEol}}{{exit 1}}{{end}}' 2>/dev/null; then
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
echo ""
echo "Test 10: Non-EOL version should continue normally"
echo "Command: ./eol-test release go 1.23 -t '{{if .IsEol}}{{exit 1}}{{end}}Active: {{.Name}}'"
result=$(./eol-test release go 1.23 -t '{{if .IsEol}}{{exit 1}}{{end}}Active: {{.Name}}')
echo "Result: $result"
if [[ "$result" =~ "Active:" ]]; then
    echo "✅ PASS (non-EOL version processed normally)"
else
    echo "❌ FAIL: Expected 'Active:' in output, got '$result'"
    exit 1
fi

echo ""
echo "================================================="
echo "🎉 All inline template tests passed!"
echo ""
echo "Cleaning up..."
rm -f eol-test

echo "✅ Test completed successfully!"
