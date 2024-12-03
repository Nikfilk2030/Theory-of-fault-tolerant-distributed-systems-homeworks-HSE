cmake_minimum_required(VERSION 3.10)
project(task1)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_executable(client client.cpp)
add_executable(worker worker.cpp)

target_compile_options(client PRIVATE -Wall -Wextra)
target_compile_options(worker PRIVATE -Wall -Wextra)
