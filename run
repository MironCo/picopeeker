#!/bin/bash

set -e  # Exit on any error

echo "========================================"
echo "Running Desktop Application"
echo "========================================"
echo ""
echo "Available serial ports:"
ls /dev/tty.* 2>/dev/null || echo "No /dev/tty.* ports found"
echo ""
echo "Launching desktop application..."
echo ""

# Navigate to desktop-app and run
cd desktop-app
make run
