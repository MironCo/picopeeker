#!/bin/bash

set -e  # Exit on any error

echo "========================================"
echo "Building Raspberry Pi Pico Code"
echo "========================================"

# Create build directory if it doesn't exist
mkdir -p build

# Navigate to build directory and run CMake
cd build
cmake ..
make

echo ""
echo "✓ Pico build complete! Files in build/"
echo ""

# Go back to root
cd ..

echo "========================================"
echo "Building Desktop Application"
echo "========================================"

# Navigate to desktop-app and build
cd desktop-app
make build

echo ""
echo "✓ Desktop app build complete! Binary in desktop-app/bin/"
echo ""

echo "========================================"
echo "Build Summary"
echo "========================================"
echo "Pico firmware: build/blink.uf2"
echo "Desktop app:   desktop-app/bin/desktop-app"
echo "========================================"
echo ""
echo "Available serial ports:"
ls /dev/tty.*
echo ""
echo "Launching desktop application..."
echo ""

./bin/desktop-app
