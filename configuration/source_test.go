package configuration_test

import (
	//. "github.com/AliceO2Group/Control/configuration"
	. "github.com/onsi/ginkgo"
	//. "github.com/onsi/gomega"
	//"gopkg.in/yaml.v2"
)

var _ = Describe("Source", func() {
//	var (
//		c   Source
//		err error
//	)
//
//	DoConfigurationTests := func() {
//		var (
//			o2_control_tasks_1_map = Map{
//				"name": String("fairmq-ex-1-n-1-sampler"),
//				"control": Map{
//					"mode": String("fairmq"),
//				},
//				"wants": Map{
//					"cpu": String("1"),
//					"memory": String("256"),
//					"ports": String("1"),
//				},
//				"bind": Array{
//					Map{
//						"name": String("data1"),
//						"type": String("push"),
//						"sndBufSize": String("1000"),
//						"rcvBufSize": String("1000"),
//						"rateLogging": String("0"),
//					},
//				},
//				"properties": Map{
//					"severity": String("trace"),
//					"color": String("false"),
//				},
//				"command": Map{
//					"env": Array{},
//					"shell": String("true"),
//					"arguments": Array{},
//					"value": String("fairmq-ex-1-n-1-sampler"),
//				},
//			}
//			recursivePutMap = Map{
//				"firstKey": String("one"),
//				"secondKey": Array{
//					Map{
//						"name": String("first"),
//						"type": String("an array item"),
//					},
//					Map{
//						"name": String("second"),
//						"type": String("an array item"),
//					},
//					Map{
//						"name": String("third"),
//						"type": String("and yet another array item"),
//					},
//				},
//				"thirdKey": Map{
//					"just some": String("stuff"),
//				},
//			}
//			recursivePutArray = Array{
//				Map{
//					"name": String("first"),
//					"type": String("an array item with a property map inside"),
//					"properties": Map{
//						"just some": String("stuff"),
//					},
//				},
//				Map{
//					"name": String("second"),
//					"type": String("an array item"),
//				},
//				Map{
//					"name": String("third"),
//					"type": String("and yet another array item"),
//				},
//			}
//			recursivePutString = String("this is a bit underwhelming compared to the other two...")
//		)
//
//		It("should return no error when creating an instance", func() {
//			Expect(err).NotTo(HaveOccurred())
//		})
//
//		Context("to get a subtree or value", func() {
//			It("should correctly get a single item", func() {
//				Expect(c.Get("o2/control/globals/o2_install_path")).To(Equal("/opt/alisw/el7"))
//				Expect(c.Get("o2/control/tasks[1]/name")).To(Equal("fairmq-ex-1-n-1-sampler"))
//				Expect(c.Get("o2/control/globals/control/direct/control_port_args[1]")).To(Equal("{{ controlPort }}"))
//			})
//
//			It("should correctly recursively get a configuration subtree", func() {
//				Expect(c.GetRecursive("o2/control/tasks[1]")).To(Equal(o2_control_tasks_1_map))
//			})
//
//			It("should correctly recursively get a configuration subtree as YAML", func() {
//				marshalled, marshErr := yaml.Marshal(o2_control_tasks_1_map)
//				Expect(marshErr).NotTo(HaveOccurred())
//				Expect(c.GetRecursiveYaml("o2/control/tasks[1]")).To(Equal(marshalled))
//			})
//		})
//
//		Context("to check the existence of a key", func() {
//			It("should correctly determine whether a value node is defined", func() {
//				Expect(c.Exists("o2/control/workflows[0]/role/roles[1]/name")).To(BeTrue())
//				Expect(c.Exists("o2/control/workflows[0]/role/roles[4]/name")).To(BeFalse())
//			})
//
//			It("should correctly determine whether a map node is defined", func() {
//				Expect(c.Exists("o2")).To(BeTrue())
//			})
//
//			It("should correctly determine whether an array node is defined", func() {
//				Expect(c.Exists("o2/control/globals/control/direct/control_port_args[1]")).To(BeTrue())
//				Expect(c.Exists("o2/control/globals/control/direct/control_port_args")).To(BeTrue())
//			})
//
//			It("should correctly address non-existent keys", func() {
//				Expect(c.Exists("")).To(BeFalse())
//				Expect(c.Exists("FakeKey")).To(BeFalse())
//			})
//		})
//
//		Context("to add a new subtree or value", func(){
//			It("should correctly push a single value", func() {
//				Expect(c.Exists("o2/control/tasks[0]/bind[1]/newSingleValue")).To(BeFalse())
//				putErr := c.Put("o2/control/tasks[0]/bind[1]/newSingleValue", "foobar")
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.Get("o2/control/tasks[0]/bind[1]/newSingleValue")).To(Equal("foobar"))
//			})
//
//			It("should correctly push a configuration.Map", func() {
//				Expect(c.Exists("o2/control/tasks[0]/bind[1]/newMap")).To(BeFalse())
//				putErr := c.PutRecursive("o2/control/tasks[0]/bind[1]/newMap", recursivePutMap)
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.GetRecursive("o2/control/tasks[0]/bind[1]/newMap")).To(Equal(recursivePutMap))
//			})
//
//			It("should correctly push an configuration.Array", func() {
//				Expect(c.Exists("o2/control/tasks[0]/bind[1]/newArray")).To(BeFalse())
//				putErr := c.PutRecursive("o2/control/tasks[0]/bind[1]/newArray", recursivePutArray)
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.GetRecursive("o2/control/tasks[0]/bind[1]/newArray")).To(Equal(recursivePutArray))
//			})
//
//			It("should correctly push a configuration.String", func() {
//				Expect(c.Exists("o2/control/tasks[0]/bind[1]/newString")).To(BeFalse())
//				putErr := c.PutRecursive("o2/control/tasks[0]/bind[1]/newString", recursivePutString)
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.GetRecursive("o2/control/tasks[0]/bind[1]/newString")).To(BeEquivalentTo(recursivePutString))
//			})
//
//			It("should correctly push a YAML map", func() {
//				Expect(c.Exists("o2/control/tasks[0]/bind[1]/newYamlMap")).To(BeFalse())
//				marshalled, marshErr := yaml.Marshal(recursivePutMap)
//				Expect(marshErr).NotTo(HaveOccurred())
//				putErr := c.PutRecursiveYaml("o2/control/tasks[0]/bind[1]/newYamlMap", marshalled)
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.GetRecursiveYaml("o2/control/tasks[0]/bind[1]/newYamlMap")).To(Equal(marshalled))
//			})
//
//			It("should correctly push a YAML array", func() {
//				Expect(c.Exists("o2/control/tasks[0]/bind[1]/newYamlArray")).To(BeFalse())
//				marshalled, marshErr := yaml.Marshal(recursivePutArray)
//				Expect(marshErr).NotTo(HaveOccurred())
//				putErr := c.PutRecursiveYaml("o2/control/tasks[0]/bind[1]/newYamlArray", marshalled)
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.GetRecursiveYaml("o2/control/tasks[0]/bind[1]/newYamlArray")).To(Equal(marshalled))
//			})
//		})
//
//		Context("to replace/update an existing subtree or value", func() {
//			It("should correctly push a single value", func() {
//				Expect(c.Exists("o2/control/globals/config_basedir")).To(BeTrue())
//				putErr := c.Put("o2/control/globals/config_basedir", "foobar")
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.Get("o2/control/globals/config_basedir")).To(Equal("foobar"))
//			})
//
//			It("should correctly push a configuration.Map", func() {
//				Expect(c.Exists("o2/control/tasks[0]/bind[1]")).To(BeTrue())
//				putErr := c.PutRecursive("o2/control/tasks[0]/bind[1]", recursivePutMap)
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.GetRecursive("o2/control/tasks[0]/bind[1]")).To(Equal(recursivePutMap))
//			})
//
//			It("should correctly push an configuration.Array", func() {
//				Expect(c.Exists("o2/control/tasks[0]/wants")).To(BeTrue())
//				putErr := c.PutRecursive("o2/control/tasks[0]/wants", recursivePutArray)
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.GetRecursive("o2/control/tasks[0]/wants")).To(Equal(recursivePutArray))
//			})
//
//			It("should correctly push a configuration.String", func() {
//				Expect(c.Exists("o2/control/tasks[0]/wants")).To(BeTrue())
//				putErr := c.PutRecursive("o2/control/tasks[0]/wants", recursivePutString)
//				Expect(putErr).NotTo(HaveOccurred())
//				Expect(c.GetRecursive("o2/control/tasks[0]/wants")).To(BeEquivalentTo(recursivePutString))
//			})
//		})
//	}
//
//	Describe("when interacting with an instance", func() {
//		Context("with Consul backend", func() {
//			BeforeEach(func() {
//				c, err = NewSource("consul://dummy")
//			})
//
//			It("should be of type *ConsulSource", func() {
//				_, ok := c.(*ConsulSource)
//				Expect(ok).To(Equal(true))
//			})
//
//			//DoConfigurationTests()
//		})
//
//		Context("with YAML file backend", func() {
//			BeforeEach(func() {
//				c, err = NewSource("file://" + *tmpDir + "/" + configFile)
//			})
//
//			It("should be of type *YamlSource", func() {
//				_, ok := c.(*YamlSource)
//				Expect(ok).To(Equal(true))
//			})
//
//			DoConfigurationTests()
//		})
//	})
})
