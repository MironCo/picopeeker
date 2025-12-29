cmake_minimum_required(VERSION 3.13)

set(PICO_BOARD pico2)

include(pico_sdk_import.cmake)

project(picopeeker_example C CXX ASM)

set(CMAKE_C_STANDARD 11)
set(CMAKE_CXX_STANDARD 17)

pico_sdk_init()

# Example project showing PicoPeeker integration
add_executable(picopeeker_example
    example.c
)

pico_add_extra_outputs(picopeeker_example)

target_link_libraries(picopeeker_example
    pico_stdlib
    pico_multicore
)
pico_enable_stdio_usb(picopeeker_example 1)
pico_enable_stdio_uart(picopeeker_example 0)