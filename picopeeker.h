/*
 * PicoPeeker - Header-Only Memory Inspection Library for Raspberry Pi Pico
 *
 * Drop-in library for debugging memory in real-time via serial.
 * Runs on Core 1 to avoid interfering with your application.
 *
 * Usage:
 *   #include "picopeeker.h"
 *
 *   int main() {
 *       stdio_init_all();
 *       picopeeker_start();  // Launches on Core 1
 *
 *       // Your application code here
 *       while(1) {
 *           // ...
 *       }
 *   }
 *
 * Commands (sent via serial):
 *   READ:0xADDRESS:LENGTH   - Read memory region
 *   SEARCH:HEXPATTERN       - Search SRAM for pattern
 *   SEARCHFLASH:HEXPATTERN  - Search Flash for pattern
 *   LANDMARKS               - Show memory landmarks
 *
 * NOTE: Reads memory while app is running - may see transient values during updates.
 *       This is normal for a debugging tool. Flash is always safe (read-only).
 *
 * License: MIT
 */

#ifndef PICOPEEKER_H
#define PICOPEEKER_H

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <stdint.h>
#include "pico/stdlib.h"
#include "pico/multicore.h"

#ifdef __cplusplus
extern "C" {
#endif

// Configuration
#ifndef PICOPEEKER_CMD_BUFFER_SIZE
#define PICOPEEKER_CMD_BUFFER_SIZE 128
#endif

#ifndef PICOPEEKER_MAX_PATTERN_SIZE
#define PICOPEEKER_MAX_PATTERN_SIZE 64
#endif

#ifndef PICOPEEKER_MAX_READ_SIZE
#define PICOPEEKER_MAX_READ_SIZE 4096
#endif

#ifndef PICOPEEKER_MAX_SEARCH_RESULTS
#define PICOPEEKER_MAX_SEARCH_RESULTS 100
#endif

// Memory regions for Pico 2 (RP2350)
#define PICOPEEKER_ROM_START      0x00000000
#define PICOPEEKER_ROM_END        0x00004000
#define PICOPEEKER_FLASH_START    0x10000000
#define PICOPEEKER_FLASH_END      0x10400000  // 4MB
#define PICOPEEKER_SRAM_START     0x20000000
#define PICOPEEKER_SRAM_END       0x20082000  // 520KB
#define PICOPEEKER_PERIPH_START   0x40000000
#define PICOPEEKER_PERIPH_END     0x60000000

// Internal state
static struct {
    bool initialized;
    char cmd_buffer[PICOPEEKER_CMD_BUFFER_SIZE];
    int cmd_index;
} _picopeeker_state = {0};

// Forward declarations
static void _picopeeker_send_hex_dump(uint32_t address, uint32_t length);
static void _picopeeker_send_landmarks(void);
static void _picopeeker_search_region(uint32_t start_addr, uint32_t end_addr,
                                      const char* region_name, uint8_t* pattern, size_t pattern_len);
static void _picopeeker_search_memory(uint8_t* pattern, size_t pattern_len,
                                      bool search_flash, bool search_sram);
static void _picopeeker_parse_command(char* cmd);
static void _picopeeker_core1_main(void);

// Implementation

static void _picopeeker_send_hex_dump(uint32_t address, uint32_t length) {
    uint8_t* ptr = (uint8_t*)address;

    printf("=== HEX DUMP ===\n");
    printf("Address: 0x%08x, Length: %u bytes\n\n", address, length);

    // Print header
    printf("Address:  00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F  ASCII\n");
    printf("--------  -----------------------------------------------  ----------------\n");

    for(uint32_t i = 0; i < length; i += 16) {
        // Print address
        printf("%08x: ", address + i);

        // Print hex bytes
        for(int j = 0; j < 16; j++) {
            if(i + j < length) {
                printf("%02x ", ptr[i + j]);
            } else {
                printf("   ");
            }
        }

        printf(" ");

        // Print ASCII
        for(int j = 0; j < 16 && (i + j) < length; j++) {
            uint8_t c = ptr[i + j];
            if(c >= 32 && c <= 126) {
                printf("%c", c);
            } else {
                printf(".");
            }
        }

        printf("\n");
        fflush(stdout);
    }

    printf("\n===END===\n");
    fflush(stdout);
}

static void _picopeeker_send_landmarks(void) {
    // Extern reference to main if it exists
    extern int main(void);

    printf("LANDMARKS:\n");
    printf("main=0x%08x\n", (unsigned int)main);
    printf("END_LANDMARKS\n\n");
    fflush(stdout);
}

static void _picopeeker_search_region(uint32_t start_addr, uint32_t end_addr,
                                      const char* region_name, uint8_t* pattern, size_t pattern_len) {
    uint8_t* ptr = (uint8_t*)start_addr;
    uint32_t total_size = end_addr - start_addr;
    int found_count = 0;

    printf("=== SEARCHING %s ===\n", region_name);
    printf("Range: 0x%08x - 0x%08x (%u bytes)\n", start_addr, end_addr, total_size);
    printf("Pattern: ");
    for(size_t i = 0; i < pattern_len; i++) {
        printf("%02x ", pattern[i]);
    }
    printf("(%zu bytes)\n\n", pattern_len);
    fflush(stdout);

    // Search through memory region
    for(uint32_t offset = 0; offset <= total_size - pattern_len; offset++) {
        bool match = true;
        for(size_t i = 0; i < pattern_len; i++) {
            if(ptr[offset + i] != pattern[i]) {
                match = false;
                break;
            }
        }

        if(match) {
            uint32_t found_addr = start_addr + offset;
            printf("FOUND: 0x%08x\n", found_addr);
            found_count++;
            fflush(stdout);

            // Limit results to prevent overwhelming output
            if(found_count >= PICOPEEKER_MAX_SEARCH_RESULTS) {
                printf("(stopping after %d matches)\n", PICOPEEKER_MAX_SEARCH_RESULTS);
                fflush(stdout);
                break;
            }
        }
    }

    printf("Total matches in %s: %d\n\n", region_name, found_count);
    fflush(stdout);
}

static void _picopeeker_search_memory(uint8_t* pattern, size_t pattern_len,
                                      bool search_flash, bool search_sram) {
    if(search_flash) {
        _picopeeker_search_region(PICOPEEKER_FLASH_START, PICOPEEKER_FLASH_END,
                                  "FLASH", pattern, pattern_len);
    }

    if(search_sram) {
        _picopeeker_search_region(PICOPEEKER_SRAM_START, PICOPEEKER_SRAM_END,
                                  "SRAM", pattern, pattern_len);
    }

    printf("===END===\n");
    fflush(stdout);
}

static void _picopeeker_parse_command(char* cmd) {
    // Handle LANDMARKS command
    if(strcmp(cmd, "LANDMARKS") == 0) {
        _picopeeker_send_landmarks();
        return;
    }

    // Save original cmd for later parsing
    char cmd_copy[PICOPEEKER_CMD_BUFFER_SIZE];
    strncpy(cmd_copy, cmd, PICOPEEKER_CMD_BUFFER_SIZE - 1);
    cmd_copy[PICOPEEKER_CMD_BUFFER_SIZE - 1] = '\0';

    char* token = strtok(cmd, ":");
    if(token == NULL) {
        printf("ERROR: Invalid command\n");
        fflush(stdout);
        return;
    }

    // Handle SEARCH command (SRAM)
    if(strcmp(token, "SEARCH") == 0) {
        token = strtok(NULL, ":");
        if(token == NULL) {
            printf("ERROR: Missing search pattern\n");
            printf("Usage: SEARCH:HEXPATTERN\n");
            printf("Example: SEARCH:DEADBEEF\n");
            fflush(stdout);
            return;
        }

        // Parse hex pattern
        size_t pattern_len = strlen(token);
        if(pattern_len % 2 != 0) {
            printf("ERROR: Hex pattern must have even number of digits\n");
            fflush(stdout);
            return;
        }

        pattern_len /= 2;  // Convert to byte length
        if(pattern_len == 0 || pattern_len > PICOPEEKER_MAX_PATTERN_SIZE) {
            printf("ERROR: Pattern length must be 1-%d bytes\n", PICOPEEKER_MAX_PATTERN_SIZE);
            fflush(stdout);
            return;
        }

        uint8_t pattern[PICOPEEKER_MAX_PATTERN_SIZE];
        for(size_t i = 0; i < pattern_len; i++) {
            char hex_byte[3] = {token[i*2], token[i*2+1], '\0'};
            pattern[i] = (uint8_t)strtoul(hex_byte, NULL, 16);
        }

        _picopeeker_search_memory(pattern, pattern_len, false, true);
        return;
    }

    // Handle SEARCHFLASH command
    if(strcmp(token, "SEARCHFLASH") == 0) {
        token = strtok(NULL, ":");
        if(token == NULL) {
            printf("ERROR: Missing search pattern\n");
            printf("Usage: SEARCHFLASH:HEXPATTERN\n");
            printf("Example: SEARCHFLASH:48656C6C6F (search for 'Hello')\n");
            fflush(stdout);
            return;
        }

        // Parse hex pattern
        size_t pattern_len = strlen(token);
        if(pattern_len % 2 != 0) {
            printf("ERROR: Hex pattern must have even number of digits\n");
            fflush(stdout);
            return;
        }

        pattern_len /= 2;  // Convert to byte length
        if(pattern_len == 0 || pattern_len > PICOPEEKER_MAX_PATTERN_SIZE) {
            printf("ERROR: Pattern length must be 1-%d bytes\n", PICOPEEKER_MAX_PATTERN_SIZE);
            fflush(stdout);
            return;
        }

        uint8_t pattern[PICOPEEKER_MAX_PATTERN_SIZE];
        for(size_t i = 0; i < pattern_len; i++) {
            char hex_byte[3] = {token[i*2], token[i*2+1], '\0'};
            pattern[i] = (uint8_t)strtoul(hex_byte, NULL, 16);
        }

        _picopeeker_search_memory(pattern, pattern_len, true, false);
        return;
    }

    // Handle READ command
    if(strcmp(token, "READ") != 0) {
        printf("ERROR: Invalid command\n");
        fflush(stdout);
        return;
    }

    // Get address
    token = strtok(NULL, ":");
    if(token == NULL) {
        printf("ERROR: Missing address\n");
        fflush(stdout);
        return;
    }
    uint32_t address = strtoul(token, NULL, 16);

    // Get length
    token = strtok(NULL, ":");
    if(token == NULL) {
        printf("ERROR: Missing length\n");
        fflush(stdout);
        return;
    }
    uint32_t length = atoi(token);

    // Validate length
    if(length == 0 || length > PICOPEEKER_MAX_READ_SIZE) {
        printf("ERROR: Length must be 1-%d\n", PICOPEEKER_MAX_READ_SIZE);
        fflush(stdout);
        return;
    }

    // Validate address range and clamp length
    uint32_t max_length = length;
    uint32_t region_end = 0;

    if(address >= PICOPEEKER_ROM_START && address < PICOPEEKER_ROM_END) {
        region_end = PICOPEEKER_ROM_END;
    } else if(address >= PICOPEEKER_FLASH_START && address < PICOPEEKER_FLASH_END) {
        region_end = PICOPEEKER_FLASH_END;
    } else if(address >= PICOPEEKER_SRAM_START && address < PICOPEEKER_SRAM_END) {
        region_end = PICOPEEKER_SRAM_END;
    } else if(address >= PICOPEEKER_PERIPH_START && address < PICOPEEKER_PERIPH_END) {
        region_end = PICOPEEKER_PERIPH_END;
    } else {
        printf("ERROR: Address out of valid range\n");
        printf("Valid ranges:\n");
        printf("  ROM:         0x%08x-0x%08x\n", PICOPEEKER_ROM_START, PICOPEEKER_ROM_END - 1);
        printf("  Flash:       0x%08x-0x%08x\n", PICOPEEKER_FLASH_START, PICOPEEKER_FLASH_END - 1);
        printf("  SRAM:        0x%08x-0x%08x\n", PICOPEEKER_SRAM_START, PICOPEEKER_SRAM_END - 1);
        printf("  Peripherals: 0x%08x-0x%08x\n", PICOPEEKER_PERIPH_START, PICOPEEKER_PERIPH_END - 1);
        fflush(stdout);
        return;
    }

    // Clamp length to not exceed region boundary
    if(address + length > region_end) {
        max_length = region_end - address;
        printf("WARNING: Length clamped from %u to %u bytes to stay within region bounds\n",
               length, max_length);
        fflush(stdout);
    }

    _picopeeker_send_hex_dump(address, max_length);
}

static void _picopeeker_core1_main(void) {
    printf("PicoPeeker ready!\n");
    printf("Commands:\n");
    printf("  READ:0xADDRESS:LENGTH   - Read memory\n");
    printf("  SEARCH:HEXPATTERN       - Search SRAM for hex pattern\n");
    printf("  SEARCHFLASH:HEXPATTERN  - Search Flash for hex pattern\n");
    printf("  LANDMARKS               - Show memory landmarks\n");
    printf("Examples:\n");
    printf("  READ:0x20000000:256\n");
    printf("  SEARCH:2A000000 (search for int 42 in SRAM)\n");
    printf("  SEARCHFLASH:48656C6C6F (search for 'Hello' in Flash)\n\n");
    fflush(stdout);

    // Send landmarks on startup
    _picopeeker_send_landmarks();

    while(true) {
        int c = getchar_timeout_us(0);

        if(c != PICO_ERROR_TIMEOUT) {
            if(c == '\n' || c == '\r') {
                // Command complete
                if(_picopeeker_state.cmd_index > 0) {
                    _picopeeker_state.cmd_buffer[_picopeeker_state.cmd_index] = '\0';
                    _picopeeker_parse_command(_picopeeker_state.cmd_buffer);
                    _picopeeker_state.cmd_index = 0;
                }
            } else if(_picopeeker_state.cmd_index < PICOPEEKER_CMD_BUFFER_SIZE - 1) {
                _picopeeker_state.cmd_buffer[_picopeeker_state.cmd_index++] = (char)c;
            }
        }

        sleep_ms(1);
    }
}

// Public API

/**
 * Start PicoPeeker on Core 1
 * Call this once from your main() after stdio_init_all()
 * PicoPeeker will run autonomously in the background
 */
static inline void picopeeker_start(void) {
    if(_picopeeker_state.initialized) {
        return;  // Already running
    }

    _picopeeker_state.initialized = true;
    _picopeeker_state.cmd_index = 0;

    // Launch on Core 1
    multicore_launch_core1(_picopeeker_core1_main);
}

#ifdef __cplusplus
}
#endif

#endif // PICOPEEKER_H
