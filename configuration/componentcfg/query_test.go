package componentcfg_test

import (
	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

var _ = Describe("query", func() {
	Describe("Component configuration query", func() {
		var (
			q   *componentcfg.Query
			err error
		)

		When("creating a new query with the full path and timestamp", func() {
			BeforeEach(func() {
				q, err = componentcfg.NewQuery("qc/PHYSICS/pp/ctp-raw-qc@1234")
			})
			It("should be parsed without reporting errors", func() {
				Expect(err).To(BeNil())
			})
			It("should return the path correctly", func() {
				Expect(q.Path()).To(Equal("qc/PHYSICS/pp/ctp-raw-qc@1234"))
			})
			It("should return the raw query correctly", func() {
				Expect(q.Raw()).To(Equal("qc/PHYSICS/pp/ctp-raw-qc/1234"))
			})
			It("should return the query without the timestamp correctly", func() {
				Expect(q.WithoutTimestamp()).To(Equal("qc/PHYSICS/pp/ctp-raw-qc"))
			})
			It("should return the absolute raw query correctly", func() {
				Expect(q.AbsoluteRaw()).To(Equal(componentcfg.ConfigComponentsPath + "qc/PHYSICS/pp/ctp-raw-qc/1234"))
			})
			It("should return the query without the timestamp correctly", func() {
				Expect(q.AbsoluteWithoutTimestamp()).To(Equal(componentcfg.ConfigComponentsPath + "qc/PHYSICS/pp/ctp-raw-qc"))
			})
			It("should be able to generalize to a query for any run type", func() {
				Expect(q.WithFallbackRunType().Path()).To(Equal("qc/ANY/pp/ctp-raw-qc@1234"))
			})
			It("should be able to generalize to a query for any role name", func() {
				Expect(q.WithFallbackRunType().Path()).To(Equal("qc/ANY/pp/ctp-raw-qc@1234"))
			})
		})

		When("creating a new query with the full path but no timestamp", func() {
			BeforeEach(func() {
				q, err = componentcfg.NewQuery("qc/PHYSICS/pp/ctp-raw-qc")
			})
			It("should be parsed without reporting errors", func() {
				Expect(err).To(BeNil())
			})
			It("should return the path correctly", func() {
				Expect(q.Path()).To(Equal("qc/PHYSICS/pp/ctp-raw-qc"))
			})
			It("should return the raw query correctly", func() {
				Expect(q.Raw()).To(Equal("qc/PHYSICS/pp/ctp-raw-qc"))
			})
			It("should return the query without the timestamp correctly", func() {
				Expect(q.WithoutTimestamp()).To(Equal("qc/PHYSICS/pp/ctp-raw-qc"))
			})
		})

		When("creating a new query with all the supported types of characters", func() {
			BeforeEach(func() {
				q, err = componentcfg.NewQuery("aAzZ09-_/ANY/aAzZ09-_/aAzZ09-_")
			})
			It("should be parsed without reporting errors", func() {
				Expect(err).To(BeNil())
			})
		})

		Describe("dealing with incorrectly formatted queries", func() {
			When("query is empty", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQuery("")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			When("query has not enough / separators", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQuery("qc/ANY/any")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			When("query has an empty token", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQuery("qc/ANY//ctp-raw-qc")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			When("query has a timestamp separator, but the timestamp itself is missing", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQuery("qc/ANY/any/ctp-raw-qc@")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			// I don't know whether we are expected to support it or not, but the current behaviour is that we don't,
			// even though one can write a key with a space to consul
			When("query has a space in the entry key", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQuery("qc/ANY/any/ctp qc")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			When("run type uses lower case letters", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQuery("qc/physics/any/ctp-raw-qc")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			When("run type is unknown", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQuery("qc/FOO/any/ctp-raw-qc")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
		})
	})

	Describe("Component configuration entries query", func() {
		var (
			q   *componentcfg.EntriesQuery
			err error
		)

		When("creating a new valid query", func() {
			BeforeEach(func() {
				q, err = componentcfg.NewEntriesQuery("qc/PHYSICS/pp")
			})
			It("should be parsed without reporting errors", func() {
				Expect(err).To(BeNil())
			})
			It("should have parsed the component correctly", func() {
				Expect(q.Component).To(Equal("qc"))
			})
			It("should have parsed the run type correctly", func() {
				Expect(q.RunType).To(Equal(apricotpb.RunType_PHYSICS))
			})
			It("should have parsed the role name correctly", func() {
				Expect(q.RoleName).To(Equal("pp"))
			})
		})

		When("creating a new query with all the supported types of characters", func() {
			BeforeEach(func() {
				q, err = componentcfg.NewEntriesQuery("aAzZ09-_/ANY/aAzZ09-_")
			})
			It("should be parsed without reporting errors", func() {
				Expect(err).To(BeNil())
			})
		})

		Describe("dealing with incorrectly formatted queries", func() {
			When("query is empty", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewEntriesQuery("")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			When("query has not enough / separators", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewEntriesQuery("qc/ANY")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			When("query has an empty token", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewEntriesQuery("qc/ANY/")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
		})
	})

	Describe("Component configuration query parameters", func() {
		var (
			q   *componentcfg.QueryParameters
			err error
		)

		When("creating new valid query parameters", func() {
			BeforeEach(func() {
				q, err = componentcfg.NewQueryParameters("process=true&a=aaa&b=123&C_D3=C,C,C&detectors=[\"MCH\",\"MID\"]")
			})
			It("should be parsed without reporting errors", func() {
				Expect(err).To(BeNil())
			})
			It("should have parsed the process variable correctly", func() {
				Expect(q.ProcessTemplates).To(BeTrue())
			})
			It("should have parsed the var stack correctly", func() {
				Expect(q.VarStack["a"]).To(Equal("aaa"))
				Expect(q.VarStack["b"]).To(Equal("123"))
				Expect(q.VarStack["C_D3"]).To(Equal("C,C,C"))
				Expect(q.VarStack["detectors"]).To(Equal("[\"MCH\",\"MID\"]"))
			})
		})

		Describe("dealing with incorrectly formatted query parameters", func() {
			When("query parameters are empty", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQueryParameters("")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			When("a key has empty value", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQueryParameters("process=")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
			When("there are two identical keys", func() {
				BeforeEach(func() {
					q, err = componentcfg.NewQueryParameters("a=33&a=34")
				})
				It("should return an error", func() {
					Expect(err).To(MatchError(componentcfg.E_BAD_KEY))
				})
			})
		})
	})
})

func TestQuery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Query Suite")
}
