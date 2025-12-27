# PicoPeeker

A memory exploration tool for the Raspberry Pi Pico 2 (RP2350), featuring a desktop GUI application for interactive memory inspection and searching.

## Features

- **Memory Reading**: Read and inspect memory from ROM, Flash, SRAM, and peripheral regions
- **Memory Search**: Search through SRAM for hex patterns, ASCII strings, or 32-bit integers
- **Desktop GUI**: Clean, responsive interface built with Fyne for Go
- **Automatic Port Detection**: Finds your Pico automatically on macOS/Linux
- **Safety Features**:
  - Bounds checking to prevent reading past memory boundaries
  - Self-referential detection for search results
  - Input validation for addresses and patterns

## Hardware Requirements

- **Raspberry Pi Pico 2** (RP2350)
  - 520KB SRAM
  - Dual Cortex-M33 cores OR dual RISC-V Hazard3 cores

> **Note**: This tool is specifically designed for the Pico 2 (RP2350). It will not work correctly with the original Pico (RP2040) due to different memory sizes.

## Components

### Pico Firmware (`memory_explorer.c`)
C firmware that runs on the RP2350 and provides:
- Serial command interface for memory operations
- Hex dump output with ASCII representation
- SRAM search functionality
- Memory landmarks reporting

### Desktop Application (`desktop-app/`)
Go/Fyne GUI application that:
- Connects to the Pico via USB serial
- Provides tabbed interface for reading and searching memory
- Displays results in a monospace, copyable text area
- Supports multiple search types (hex bytes, ASCII strings, integers)

## Memory Map (RP2350)

- **ROM**: `0x00000000 - 0x00003FFF` (16KB)
- **Flash**: `0x10000000 - 0x103FFFFF` (4MB)
- **SRAM**: `0x20000000 - 0x20081FFF` (520KB)
- **Peripherals**: `0x40000000 - 0x5FFFFFFF`

## Building

### Prerequisites

- Pico SDK (configured for RP2350/Pico 2)
- CMake
- Go 1.16+ (for desktop app)
- Fyne dependencies (see [Fyne Getting Started](https://fyne.io/))

### Pico Firmware

```bash
./build.sh
```

This builds the firmware and creates a `.uf2` file in the `build/` directory. Flash it to your Pico 2 by:
1. Hold BOOTSEL while plugging in the Pico
2. Copy `build/picopeeker.uf2` to the mounted drive

### Desktop Application

```bash
cd desktop-app
go build -o bin/desktop-app main.go
```

Or use the provided build script which builds everything:
```bash
./build.sh
```

## Usage

1. Flash the firmware to your Pico 2
2. Connect the Pico via USB
3. Run the desktop application
4. The application will auto-detect your Pico's serial port
5. Use the "Read Memory" tab to inspect specific addresses
6. Use the "Search Memory" tab to find patterns in SRAM

### Reading Memory
- Enter a hex address (e.g., `0x20000000`)
- Specify the number of bytes to read (1-4096)
- Use quick access buttons for ROM, Flash, SRAM, or GPIO regions
- Navigate with +/- buttons to move through memory (256 byte increments)

### Searching Memory
- Choose search type: Hex Bytes, ASCII String, or 32-bit Int (LE)
- Enter your search pattern
- Results show all matching addresses in SRAM (up to 100 matches)
- Self-referential matches (in command buffer) are marked

## Commands

The Pico firmware accepts these serial commands:

- `LANDMARKS` - Get memory addresses of key symbols
- `READ:0xADDRESS:LENGTH` - Read memory at address
- `SEARCH:HEXPATTERN` - Search SRAM for hex pattern

## License

MIT
