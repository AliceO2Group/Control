package template

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DPL utilities", func() {
	Describe("extracting config URIs from DPL commands", func() {
		var uris []string
		When("the command does not have a config URI", func() {
			BeforeEach(func() {
				uris = extractConfigURIs("myexe --arg val")
			})
			It("should return an empty slice", func() {
				Expect(uris).To(HaveLen(0))
			})
		})
		When("URI is not complete", func() {
			BeforeEach(func() {
				uris = extractConfigURIs("myexe --config apricot://")
			})
			It("should return an empty slice", func() {
				Expect(uris).To(HaveLen(0))
			})
		})
		When("the URI is the last argument", func() {
			BeforeEach(func() {
				uris = extractConfigURIs("myexe --config apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw")
			})
			It("should return correctly parsed URIs", func() {
				Expect(uris).To(HaveLen(1))
				Expect(uris[0]).To(Equal("apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw"))
			})
		})
		When("there is a pipe adjacent to the URI", func() {
			BeforeEach(func() {
				uris = extractConfigURIs("myexe --config apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw| another-exe")
			})
			It("should return correctly parsed URIs", func() {
				Expect(uris).To(HaveLen(1))
				Expect(uris[0]).To(Equal("apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw"))
			})
		})
		When("there is a space after the URI", func() {
			BeforeEach(func() {
				uris = extractConfigURIs("myexe --config apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw --run-faster")
			})
			It("should return correctly parsed URIs", func() {
				Expect(uris).To(HaveLen(1))
				Expect(uris[0]).To(Equal("apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw"))
			})
		})
		When("the URI had a single quote at the end which is used to write a value for a variable", func() {
			BeforeEach(func() {
				uris = extractConfigURIs("myexe --config apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw?a='b' --run-faster")
			})
			It("should return correctly parsed URIs", func() {
				Expect(uris).To(HaveLen(1))
				Expect(uris[0]).To(Equal("apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw?a='b'"))
			})
		})
		When("the URI had a double quote at the end which is used to write a value for a variable", func() {
			BeforeEach(func() {
				uris = extractConfigURIs("myexe --config apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw?a=\"b\" --run-faster")
			})
			It("should return correctly parsed URIs", func() {
				Expect(uris).To(HaveLen(1))
				Expect(uris[0]).To(Equal("apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw?a=\"b\""))
			})
		})
		When("the URI is escaped with single quotes", func() {
			BeforeEach(func() {
				uris = extractConfigURIs("myexe --config 'apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw?a=[\"b\"]' --run-faster")
			})
			It("should return correctly parsed URIs", func() {
				Expect(uris).To(HaveLen(1))
				Expect(uris[0]).To(Equal("apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw?a=[\"b\"]"))
			})
		})
		When("the URI is escaped and a pipe is just after", func() {
			BeforeEach(func() {
				uris = extractConfigURIs("myexe --config 'apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw?a=[\"b\"]'| another-exe")
			})
			It("should return correctly parsed URIs", func() {
				Expect(uris).To(HaveLen(1))
				Expect(uris[0]).To(Equal("apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw?a=[\"b\"]"))
			})
		})
		When("there are two URIs", func() {
			BeforeEach(func() {
				uris = extractConfigURIs(
					"myexe --config apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw" +
						" | another-exe --config consul-json://host.cern.ch:12345/components/qc/ANY/any/tcp-raw")
			})
			It("should return correctly parsed URIs", func() {
				Expect(uris).To(HaveLen(2))
				Expect(uris[0]).To(Equal("apricot://host.cern.ch:12345/components/qc/ANY/any/ctp-raw"))
				Expect(uris[1]).To(Equal("consul-json://host.cern.ch:12345/components/qc/ANY/any/tcp-raw"))
			})
		})
	})
})
