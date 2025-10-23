cmake_minimum_required(VERSION 3.16)
project(hanfe LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

add_executable(hanfe
    src/main.cpp
    src/cli.cpp
    src/config.cpp
    src/composer.cpp
    src/layout.cpp
    src/emitter.c
    src/engine.cpp
    src/device.cpp
    src/util.cpp
)

target_include_directories(hanfe
    PRIVATE
        include
)

target_compile_options(hanfe PRIVATE -Wall -Wextra -Wpedantic)

install(TARGETS hanfe RUNTIME DESTINATION bin)
