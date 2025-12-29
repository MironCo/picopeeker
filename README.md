# PicoPeeker

A header-only memory inspection library for Raspberry Pi Pico 2 (RP2350). Drop into any project to debug memory in real-time via a desktop GUI.

## Quick Start

### 1. Install PicoPeeker

```bash
cd your-pico-project/
wget https://raw.githubusercontent.com/yourusername/picopeeker/main/picopeeker.h
```

### 2. Add to your code

```c
#include "picopeeker.h"

int main() {
    stdio_init_all();
    picopeeker_start();  // Launches on Core 1

    // Your code runs normally on Core 0
    while(1) {
        // Your game, sensor code, whatever
    }
}
```

### 3. Update CMakeLists.txt

```cmake
target_link_libraries(your_project
    pico_stdlib
    pico_multicore  # Add this line
    # ... your other libraries
)
```

### 4. Use the desktop GUI

Download and run the desktop app (see [Desktop Application](#desktop-application) below). It will auto-detect your Pico and let you inspect memory while your code runs.

## Features

- **Header-only**: Single file, zero build complexity
- **Non-invasive**: Runs on Core 1, won't block your app
- **Real-time inspection**: Read memory while your code runs
- **LED feedback**: Flashes onboard LED when processing commands
- **Memory search**: Find patterns in Flash and SRAM
- **Safe**: Bounds checking, input validation

## Commands

PicoPeeker responds to these serial commands:

- `READ:0xADDRESS:LENGTH` - Read memory region
- `SEARCH:HEXPATTERN` - Search SRAM for hex pattern
- `SEARCHFLASH:HEXPATTERN` - Search Flash for hex pattern
- `LANDMARKS` - Show memory addresses of key symbols

## Hardware Requirements

- **Raspberry Pi Pico 2** (RP2350)
  - 520KB SRAM
  - Dual Cortex-M33 cores OR dual RISC-V Hazard3 cores

> **Note**: Designed for Pico 2 (RP2350). Not compatible with original Pico (RP2040) due to different memory sizes and single-core architecture.

## Memory Map (RP2350)

- **ROM**: `0x00000000 - 0x00003FFF` (16KB)
- **Flash**: `0x10000000 - 0x103FFFFF` (4MB)
- **SRAM**: `0x20000000 - 0x20081FFF` (520KB)
- **Peripherals**: `0x40000000 - 0x5FFFFFFF`

## Configuration

Customize PicoPeeker by defining these before including the header:

```c
#define PICOPEEKER_LED_PIN 13              // Use different LED pin
#define PICOPEEKER_MAX_READ_SIZE 8192       // Allow larger reads
#define PICOPEEKER_MAX_SEARCH_RESULTS 200   // Show more search results

#include "picopeeker.h"
```

Available options:
- `PICOPEEKER_CMD_BUFFER_SIZE` (default: 128)
- `PICOPEEKER_MAX_PATTERN_SIZE` (default: 64)
- `PICOPEEKER_MAX_READ_SIZE` (default: 4096)
- `PICOPEEKER_MAX_SEARCH_RESULTS` (default: 100)
- `PICOPEEKER_LED_PIN` (default: 25 - onboard LED)

## Desktop Application

### Building the GUI

```bash
cd desktop-app
go build -o bin/picopeeker-gui main.go
```

Or download pre-built binaries from the [Releases](https://github.com/yourusername/picopeeker/releases) page.

### Prerequisites

- Go 1.16+ (for building from source)
- Fyne dependencies (see [Fyne Getting Started](https://fyne.io/))

### Using the GUI

1. Flash your Pico with code that includes PicoPeeker
2. Connect Pico via USB
3. Run the desktop application
4. GUI will auto-detect your Pico's serial port
5. Use the "Read Memory" tab to inspect specific addresses
6. Use the "Search Memory" tab to find patterns

#### Reading Memory
- Enter hex address (e.g., `0x20000000`)
- Specify bytes to read (1-4096)
- Quick access buttons for ROM, Flash, SRAM, GPIO
- Navigate with +/- buttons (256 byte increments)

#### Searching Memory
- Choose: Hex Bytes, ASCII String, or 32-bit Int (LE)
- Results show all matching addresses (up to 100 matches)

## Example Project

See [example.c](example.c) for a minimal integration example. Build it:

```bash
./build.sh
```

Flash `build/picopeeker_example.uf2` to your Pico, then connect the desktop GUI.

## Notes on Race Conditions

PicoPeeker reads memory while your application runs on Core 0. This means:

- **Flash reads are safe**: Flash is read-only at runtime
- **SRAM reads show snapshots**: You might see transient values during updates
- **This is normal for debugging**: Real debuggers have the same behavior

For 99% of use cases, this is fine. You're inspecting state, not doing real-time control.

## Use Cases

- Game development (inspect sprite data, collision variables)
- Verify code is executing from Flash (XIP)
- Find memory leaks
- Debug buffer overruns
- Learn C memory layouts
- Reverse engineer binary protocols

## License

MIT
