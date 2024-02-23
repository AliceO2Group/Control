find_package(Boost CONFIG COMPONENTS unit_test_framework REQUIRED)

add_executable(o2-kaki-tests)
target_sources(o2-kaki-tests PUBLIC mainTest.cpp proto.cpp simpleConsume.cpp)
target_compile_features(o2-kaki-tests PRIVATE cxx_std_20)
target_link_libraries(o2-kaki-tests PRIVATE Boost::unit_test_framework o2-kaki-protos)
target_compile_definitions(o2-kaki-tests PRIVATE BOOST_TEST_DYN_LINK)

enable_testing()
add_test(o2-kaki-tests o2-kaki-tests)
