#!/bin/bash

# 1. Run tests and generate coverage data
go test -coverprofile=coverage.out ./...

# 2. Convert coverage data to HTML
go tool cover -html=coverage.out -o coverage.html

# 3. Automatically open in your Windows browser (WSL friendly)
if command -v explorer.exe >/dev/null; then
    explorer.exe coverage.html
elif command -v xdg-open >/dev/null; then
    xdg-open coverage.html
else
    echo "Report generated! Open coverage.html in your browser."
fi
