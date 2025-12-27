cmake_minimum_required(VERSION 3.13)

set(PICO_BOARD pico2)

include(pico_sdk_import.cmake)

project(picopeeker C CXX ASM)

set(CMAKE_C_STANDARD 11)
set(CMAKE_CXX_STANDARD 17)

pico_sdk_init()

add_executable(picopeeker
    memory_explorer.c
)

pico_add_extra_outputs(picopeeker)

target_link_libraries(picopeeker pico_stdlib)
pico_enable_stdio_usb(picopeeker 1)
pico_enable_stdio_uart(picopeeker 0)