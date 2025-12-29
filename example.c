// Example: Minimal PicoPeeker integration
//
// This demonstrates how to add PicoPeeker to any Pico project.
// The debugger runs on Core 1 and doesn't interfere with your application.

#include <stdio.h>
#include "pico/stdlib.h"
#include "picopeeker.h"

int main() {
    stdio_init_all();

    // Start PicoPeeker on Core 1
    picopeeker_start();

    // Your application code runs on Core 0
    const uint LED_PIN = 25;
    gpio_init(LED_PIN);
    gpio_set_dir(LED_PIN, GPIO_OUT);

    printf("Application started! PicoPeeker is running on Core 1.\n");

    while (true) {
        gpio_put(LED_PIN, 1);
        sleep_ms(500);
        gpio_put(LED_PIN, 0);
        sleep_ms(500);
    }

    return 0;
}
