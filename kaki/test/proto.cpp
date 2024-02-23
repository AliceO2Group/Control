#include <boost/test/unit_test.hpp>

#include <test.pb.h>

BOOST_AUTO_TEST_CASE(test_simple_proto)
{
  const std::string message("bla bla");
  kaki::Test to_serialize;
  to_serialize.set_tst(message);
  const auto serialized = to_serialize.SerializeAsString();

  BOOST_TEST(!serialized.empty());

  kaki::Test parsed;
  BOOST_TEST(parsed.ParseFromString(serialized));

  BOOST_TEST(parsed.tst() == message);
  BOOST_TEST(parsed.tst() == to_serialize.tst());
}
