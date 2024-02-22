find_package(InfoLogger CONFIG NAMES InfoLogger libInfoLogger)
set_package_properties(InfoLogger PROPERTIES TYPE REQUIRED)

add_library(o2-kaki-internal STATIC)
target_sources(o2-kaki-internal PRIVATE lib/consumer.cpp)
target_include_directories(o2-kaki-internal PUBLIC lib)
target_compile_features(o2-kaki-internal PUBLIC cxx_std_20)
target_link_libraries(o2-kaki-internal PUBLIC AliceO2::InfoLogger kafka-lib)

add_executable(o2-kaki)
target_sources(o2-kaki PUBLIC app/main.cpp)
target_link_libraries(o2-kaki PRIVATE o2-kaki-internal)
