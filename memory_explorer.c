#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "pico/stdlib.h"
#include "hardware/watchdog.h"

#define CMD_BUFFER_SIZE 128

// Global variables
int global_var = 42;
int global_uninitialized;

// Forward declaration
int main(void);

void send_hex_dump(uint32_t address, uint32_t length) {
    uint8_t* ptr = (uint8_t*)address;
    
    printf("=== HEX DUMP ===\n");
    printf("Address: 0x%08x, Length: %d bytes\n\n", address, length);
    
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
        fflush(stdout);  // Flush after every line to prevent buffering
    }
    
    printf("\n===END===\n");
    fflush(stdout);
}

void send_landmarks() {
    printf("LANDMARKS:\n");
    printf("global_var=0x%08x\n", (unsigned int)&global_var);
    printf("global_uninitialized=0x%08x\n", (unsigned int)&global_uninitialized);
    printf("main=0x%08x\n", (unsigned int)main);
    printf("END_LANDMARKS\n\n");
    fflush(stdout);
}

void search_memory(uint8_t* pattern, size_t pattern_len) {
    uint32_t start_addr = 0x20000000;
    uint32_t end_addr = 0x20082000;  // 520KB SRAM
    uint8_t* ptr = (uint8_t*)start_addr;
    uint32_t total_size = end_addr - start_addr;
    int found_count = 0;

    printf("=== SEARCHING SRAM ===\n");
    printf("Range: 0x%08x - 0x%08x (%u bytes)\n", start_addr, end_addr, total_size);
    printf("Pattern: ");
    for(size_t i = 0; i < pattern_len; i++) {
        printf("%02x ", pattern[i]);
    }
    printf("(%zu bytes)\n\n", pattern_len);
    fflush(stdout);

    // Search through SRAM
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
            printf("FOUND: 0x%08x", found_addr);

            // Check if this might be self-referential (found in command buffer area)
            // Command buffer is typically near end of SRAM
            if(found_addr >= 0x20081f00 && found_addr <= 0x20081fff) {
                printf(" (Maybe Self-Referential - command buffer)");
            }

            printf("\n");
            found_count++;
            fflush(stdout);

            // Limit results to prevent overwhelming output
            if(found_count >= 100) {
                printf("(stopping after 100 matches)\n");
                fflush(stdout);
                break;
            }
        }
    }

    printf("\nTotal matches: %d\n", found_count);
    printf("===END===\n");
    fflush(stdout);
}

void parse_command(char* cmd) {
    // Handle LANDMARKS command
    if(strcmp(cmd, "LANDMARKS") == 0) {
        send_landmarks();
        return;
    }

    // Save original cmd for later parsing
    char cmd_copy[CMD_BUFFER_SIZE];
    strncpy(cmd_copy, cmd, CMD_BUFFER_SIZE - 1);
    cmd_copy[CMD_BUFFER_SIZE - 1] = '\0';

    char* token = strtok(cmd, ":");
    if(token == NULL) {
        printf("ERROR: Invalid command\n");
        fflush(stdout);
        return;
    }

    // Handle SEARCH command
    // Format: SEARCH:DEADBEEF or SEARCH:42
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
        if(pattern_len == 0 || pattern_len > 64) {
            printf("ERROR: Pattern length must be 1-64 bytes (2-128 hex digits)\n");
            fflush(stdout);
            return;
        }

        uint8_t pattern[64];
        for(size_t i = 0; i < pattern_len; i++) {
            char hex_byte[3] = {token[i*2], token[i*2+1], '\0'};
            pattern[i] = (uint8_t)strtoul(hex_byte, NULL, 16);
        }

        search_memory(pattern, pattern_len);
        return;
    }

    // Handle READ command
    // Expected format: "READ:0x20000000:256"
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
    if(length == 0 || length > 4096) {
        printf("ERROR: Length must be 1-4096\n");
        fflush(stdout);
        return;
    }

    // Validate address range and clamp length to prevent reading past boundaries
    uint32_t max_length = length;
    uint32_t region_end = 0;

    if(address >= 0x00000000 && address < 0x00004000) {  // ROM - 16KB
        region_end = 0x00004000;
    } else if(address >= 0x10000000 && address < 0x10400000) {  // Flash
        region_end = 0x10400000;
    } else if(address >= 0x20000000 && address < 0x20082000) {  // SRAM - 520KB
        region_end = 0x20082000;
    } else if(address >= 0x40000000 && address < 0x60000000) {  // Peripherals
        region_end = 0x60000000;
    } else {
        printf("ERROR: Address out of valid range\n");
        printf("Valid ranges:\n");
        printf("  ROM:         0x00000000-0x00003FFF\n");
        printf("  Flash:       0x10000000-0x103FFFFF\n");
        printf("  SRAM:        0x20000000-0x20081FFF\n");
        printf("  Peripherals: 0x40000000-0x5FFFFFFF\n");
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

    send_hex_dump(address, max_length);
}

int main() {
    stdio_init_all();
    
    // Disable watchdog to prevent random resets
    if(watchdog_caused_reboot()) {
        printf("Rebooted by watchdog!\n");
    }
    
    const uint LED_PIN = 25;
    gpio_init(LED_PIN);
    gpio_set_dir(LED_PIN, GPIO_OUT);
    
    char cmd_buffer[CMD_BUFFER_SIZE];
    int cmd_index = 0;
    
    printf("PicoPeeker ready!\n");
    printf("Commands:\n");
    printf("  READ:0xADDRESS:LENGTH - Read memory\n");
    printf("  SEARCH:HEXPATTERN     - Search SRAM for hex pattern\n");
    printf("  LANDMARKS             - Show memory landmarks\n");
    printf("Examples:\n");
    printf("  READ:0x20000000:256\n");
    printf("  SEARCH:2A000000 (search for int 42)\n");
    printf("  SEARCH:DEADBEEF\n\n");
    fflush(stdout);

    // Send landmarks on startup
    send_landmarks();
    
    while(true) {
        int c = getchar_timeout_us(0);
        
        if(c != PICO_ERROR_TIMEOUT) {
            if(c == '\n' || c == '\r') {
                // Command complete
                if(cmd_index > 0) {
                    cmd_buffer[cmd_index] = '\0';
                    
                    gpio_put(LED_PIN, 1);
                    parse_command(cmd_buffer);
                    sleep_ms(100);
                    gpio_put(LED_PIN, 0);
                    
                    cmd_index = 0;
                }
            } else if(cmd_index < CMD_BUFFER_SIZE - 1) {
                cmd_buffer[cmd_index++] = (char)c;
            }
        }
        
        sleep_ms(1);
    }
}