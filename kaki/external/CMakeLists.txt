add_subdirectory(modern-cpp-kafka)

find_package(CURL REQUIRED)

add_library(kafka-lib INTERFACE)
target_link_libraries(kafka-lib INTERFACE CURL::libcurl modern-cpp-kafka-api)
