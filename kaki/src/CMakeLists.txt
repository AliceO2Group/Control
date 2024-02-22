find_package(InfoLogger CONFIG NAMES InfoLogger libInfoLogger)
set_package_properties(InfoLogger PROPERTIES TYPE REQUIRED)

add_executable(o2-kaki)
target_sources(o2-kaki PUBLIC main.cpp)
target_compile_features(o2-kaki PRIVATE cxx_std_20)
target_link_libraries(o2-kaki PRIVATE AliceO2::InfoLogger kafka-lib)
