#!/bin/bash

set -e  # Exit on any error

# Parse command line arguments
PICO_BOARD="pico2"  # Default to Pico 2

while getopts "12h" opt; do
  case $opt in
    1)
      PICO_BOARD="pico"
      ;;
    2)
      PICO_BOARD="pico2"
      ;;
    h)
      echo "Usage: $0 [-1|-2]"
      echo "  -1  Build for Raspberry Pi Pico 1 (RP2040)"
      echo "  -2  Build for Raspberry Pi Pico 2 (RP2350) [default]"
      exit 0
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      echo "Use -h for help"
      exit 1
      ;;
  esac
done

echo "========================================"
echo "Building Raspberry Pi Pico Code"
if [ "$PICO_BOARD" = "pico" ]; then
    echo "Target: Pico 1 (RP2040)"
else
    echo "Target: Pico 2 (RP2350)"
fi
echo "========================================"

# Remove and recreate build directory for clean build
# (Required when switching between Pico 1 and Pico 2)
rm -rf build
mkdir -p build

# Navigate to build directory and run CMake
cd build
cmake -DPICO_BOARD=$PICO_BOARD ..
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
if [ "$PICO_BOARD" = "pico" ]; then
    echo "Pico firmware: build/picopeeker_example.uf2 (for Pico 1/RP2040)"
else
    echo "Pico firmware: build/picopeeker_example.uf2 (for Pico 2/RP2350)"
fi
echo "Desktop app:   desktop-app/bin/desktop-app"
echo "========================================"
echo ""
echo "Available serial ports:"
ls /dev/tty.*
echo ""
echo "Launching desktop application..."
echo ""

./bin/desktop-app
