package template_test

import (
	"encoding/json"
	"github.com/AliceO2Group/Control/apricot/local"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"strings"

	"github.com/AliceO2Group/Control/configuration/template"
)

var _ = Describe("Template function stack", func() {

	Describe("utility functions", func() {
		var utilFuncMap map[string]interface{}

		BeforeEach(func() {
			varStack := map[string]string{
				"test_var":                   "test_value",
				"prefix_var":                 "prefixed_value",
				"fallback_var":               "fallback_value",
				"prefixed_none_fallback_var": "none",
			}
			utilFuncMap = template.MakeUtilFuncMap(varStack)
		})

		Context("strings functions", func() {
			It("should validate inputs and convert string to int", func() {
				atoiFunc := utilFuncMap["Atoi"].(func(string) int)

				result := atoiFunc("123")
				Expect(result).To(Equal(123))
				// we only produce an error log if unexpected input appears
				result = atoiFunc("abc")
				Expect(result).To(Equal(0))
				result = atoiFunc("")
				Expect(result).To(Equal(0))
			})

			It("should validate inputs and convert int to string", func() {
				itoaFunc := utilFuncMap["Itoa"].(func(int) string)

				Expect(itoaFunc(123)).To(Equal("123"))
				Expect(itoaFunc(0)).To(Equal("0"))
				Expect(itoaFunc(-456)).To(Equal("-456"))
			})

			It("should trim quotes", func() {
				trimQuotesFunc := utilFuncMap["TrimQuotes"].(func(string) string)

				Expect(trimQuotesFunc("\"test\"")).To(Equal("test"))
				// Fixme: should it support also single quotes?
				//Expect(trimQuotesFunc("'test'")).To(Equal("test"))

				Expect(trimQuotesFunc("test")).To(Equal("test"))
				Expect(trimQuotesFunc("")).To(Equal(""))
				Expect(trimQuotesFunc("\"test")).To(Equal("test"))
				Expect(trimQuotesFunc("test\"")).To(Equal("test"))
			})

			It("should trim spaces", func() {
				trimSpaceFunc := utilFuncMap["TrimSpace"].(func(string) string)

				Expect(trimSpaceFunc("  test ")).To(Equal("test"))
				Expect(trimSpaceFunc("test")).To(Equal("test"))
				Expect(trimSpaceFunc("test test ")).To(Equal("test test"))
			})

			It("should convert to upper case", func() {
				toUpperFunc := utilFuncMap["ToUpper"].(func(string) string)
				Expect(toUpperFunc("test")).To(Equal("TEST"))
				Expect(toUpperFunc("Test")).To(Equal("TEST"))
				Expect(toUpperFunc("1")).To(Equal("1"))
			})

			It("should convert to lower case", func() {
				toLowerFunc := utilFuncMap["ToLower"].(func(string) string)
				Expect(toLowerFunc("TEST")).To(Equal("test"))
				Expect(toLowerFunc("Test")).To(Equal("test"))
				Expect(toLowerFunc("1")).To(Equal("1"))
			})

			It("should check if a string is truthy", func() {
				isTruthyFunc := utilFuncMap["IsTruthy"].(func(string) bool)

				Expect(isTruthyFunc("true")).To(BeTrue())
				Expect(isTruthyFunc("TRUE")).To(BeTrue())
				Expect(isTruthyFunc("yes")).To(BeTrue())
				Expect(isTruthyFunc("y")).To(BeTrue())
				Expect(isTruthyFunc("1")).To(BeTrue())
				Expect(isTruthyFunc("on")).To(BeTrue())
				Expect(isTruthyFunc("ok")).To(BeTrue())

				Expect(isTruthyFunc("false")).To(BeFalse())
				Expect(isTruthyFunc("")).To(BeFalse())
				Expect(isTruthyFunc("0")).To(BeFalse())
				Expect(isTruthyFunc("truthy")).To(BeFalse())
			})

			It("should check if a string is falsy", func() {
				isFalsyFunc := utilFuncMap["IsFalsy"].(func(string) bool)
				Expect(isFalsyFunc("false")).To(BeTrue())
				Expect(isFalsyFunc("FALSE")).To(BeTrue())
				Expect(isFalsyFunc("no")).To(BeTrue())
				Expect(isFalsyFunc("n")).To(BeTrue())
				Expect(isFalsyFunc("0")).To(BeTrue())
				Expect(isFalsyFunc("off")).To(BeTrue())
				Expect(isFalsyFunc("none")).To(BeTrue())
				Expect(isFalsyFunc("")).To(BeTrue())

				Expect(isFalsyFunc("true")).To(BeFalse())
				Expect(isFalsyFunc("1")).To(BeFalse())
			})
		})

		Context("json functions", func() {
			It("should unmarshal JSON", func() {
				unmarshalFunc := utilFuncMap["json"].(map[string]interface{})["Unmarshal"].(func(string) interface{})
				result := unmarshalFunc(`{"key": "value"}`)
				Expect(result).To(HaveKeyWithValue("key", "value"))
			})

			It("should marshal to JSON", func() {
				marshalFunc := utilFuncMap["json"].(map[string]interface{})["Marshal"].(func(interface{}) string)
				input := map[string]string{"key": "value"}
				result := marshalFunc(input)
				Expect(result).To(MatchJSON(`{"key": "value"}`))
			})
		})

		Context("uid functions", func() {
			It("should generate a new UID", func() {
				newFunc := utilFuncMap["uid"].(map[string]interface{})["New"].(func() string)
				uid1 := newFunc()
				uid2 := newFunc()
				Expect(uid1).NotTo(Equal(uid2))
			})
		})

		Context("util functions", func() {
			It("should handle prefixed override", func() {
				prefixedOverrideFunc := utilFuncMap["util"].(map[string]interface{})["PrefixedOverride"].(func(string, string) string)
				Expect(prefixedOverrideFunc("var", "prefix")).To(Equal("prefixed_value"))
				Expect(prefixedOverrideFunc("fallback_var", "nonexistent_prefix")).To(Equal("fallback_value"))
				Expect(prefixedOverrideFunc("fallback_var", "prefixed_none")).To(Equal("fallback_value"))
				Expect(prefixedOverrideFunc("nonexistent_fallback_var", "prefix")).To(Equal(""))
			})

			It("should handle nullable strings", func() {
				nullableFunc := utilFuncMap["util"].(map[string]interface{})["Nullable"].(func(*string) string)
				var nilString *string
				Expect(nullableFunc(nilString)).To(Equal(""))

				nonNilString := "test"
				Expect(nullableFunc(&nonNilString)).To(Equal("test"))
			})

			It("should check if suffix is in range", func() {
				suffixInRangeFunc := utilFuncMap["util"].(map[string]interface{})["SuffixInRange"].(func(string, string, string, string) string)
				Expect(suffixInRangeFunc("test5", "test", "1", "10")).To(Equal("true"))
				Expect(suffixInRangeFunc("test15", "test", "1", "10")).To(Equal("false"))
				// no prefix
				Expect(suffixInRangeFunc("other5", "test", "1", "10")).To(Equal("false"))
				// no suffix
				Expect(suffixInRangeFunc("test", "test", "1", "10")).To(Equal("false"))
				// range arguments are not numbers
				Expect(suffixInRangeFunc("test", "test", "a", "10")).To(Equal("false"))
				Expect(suffixInRangeFunc("test", "test", "1", "b")).To(Equal("false"))
			})
		})
	})

	Describe("inventory functions", func() {
		var (
			svc                *local.Service
			configAccessObject map[string]interface{}
			err                error
		)
		BeforeEach(func() {
			svc, err = local.NewService("file://" + *tmpDir + "/" + serviceConfigFile)
			Expect(err).NotTo(HaveOccurred())

			varStack := map[string]string{}
			configAccessObject = template.MakeConfigAccessObject(svc, varStack)
		})

		It("should get detector for host", func() {
			getDetectorFunc := configAccessObject["inventory"].(map[string]interface{})["DetectorForHost"].(func(string) string)
			Expect(getDetectorFunc("flp001")).To(Equal("ABC"))
			Expect(getDetectorFunc("NOPE")).To(ContainSubstring("error"))
		})
		It("should get detectors for a list of hosts", func() {
			getDetectorsFunc := configAccessObject["inventory"].(map[string]interface{})["DetectorsForHosts"].(func(string) string)
			Expect(getDetectorsFunc("[ \"flp001\", \"flp002\" ]")).To(Equal("[\"ABC\",\"DEF\"]"))
			Expect(getDetectorsFunc("[ \"flp001\", \"NOPE\" ]")).To(ContainSubstring("error"))
			Expect(getDetectorsFunc("[ \"NOPE\" ]")).To(ContainSubstring("error"))
			Expect(getDetectorsFunc("flp001")).To(ContainSubstring("error"))
		})
		It("should get CRU cards for host", func() {
			getCruCardsFunc := configAccessObject["inventory"].(map[string]interface{})["CRUCardsForHost"].(func(string) string)
			var result []string
			err := json.Unmarshal([]byte(getCruCardsFunc("flp001")), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ConsistOf("0228", "0229"))
			Expect(getCruCardsFunc("NOPE")).To(ContainSubstring("error"))
		})
		It("should get endpoints for CRU card", func() {
			getEndpointsFunc := configAccessObject["inventory"].(map[string]interface{})["EndpointsForCRUCard"].(func(string, string) string)
			endpoints := strings.Split(getEndpointsFunc("flp001", "0228"), " ")
			Expect(endpoints).To(ConsistOf("0", "1"))
			// fixme: probably incorrect behaviour, but I don't want to risk breaking something
			Expect(getEndpointsFunc("flp001", "NOPE")).To(BeEmpty())
		})
	})
})
