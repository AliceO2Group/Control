package local

import (
	"encoding/json"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("HTTP apricot service", func() {

	// We test the handler directly to avoid any issues with finding a free port, waiting until the server responds,
	// freeing the socket/port, concurrency of the executed tests. This way we can test just the API.
	var (
		httpSvc  *HttpService
		handler  *mux.Router
		recorder *httptest.ResponseRecorder
		err      error
	)

	Context("with YAML file backend", func() {
		BeforeEach(func() {
			svc, err := NewService("file://" + *tmpDir + "/" + serviceHTTPConfigFile)
			Expect(err).NotTo(HaveOccurred())
			httpSvc = &HttpService{svc}
			handler = newHandlerForHttpService(httpSvc)
			recorder = httptest.NewRecorder()
		})

		Describe("getting the list of hosts", func() {
			Context("expecting JSON output", func() {
				When("the list of hosts is retrieved for all detectors", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/inventory/flps/json", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should contain all the hosts for all the detectors listed in the test YAML file at o2/hardware/detectors", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))

						var hosts []string
						err = json.NewDecoder(recorder.Body).Decode(&hosts)
						Expect(err).NotTo(HaveOccurred())

						Expect(hosts).To(HaveLen(4))
						Expect(hosts).To(ContainElements("flp001", "flp002", "flp003", "flp100"))
					})
				})
				When("the list of hosts is retrieved for a valid detector", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/inventory/detectors/ITS/flps/json", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should contain all the hosts for the ITS detector in the test YAML file at o2/hardware/detectors", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))

						var hosts []string
						err = json.NewDecoder(recorder.Body).Decode(&hosts)
						Expect(err).NotTo(HaveOccurred())
						Expect(hosts).To(ContainElements("flp001"))
					})
				})
				When("the list of hosts is retrieved for a detector which is not present in the config store", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/inventory/detectors/TPC/flps/json", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					// Perhaps that's OK, but this behaviour is different from the local/service, which returns an error
					// in such case.
					It("should return an empty list", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))

						var hosts []string
						err = json.NewDecoder(recorder.Body).Decode(&hosts)
						Expect(hosts).To(BeEmpty())
					})
				})
				When("the list of hosts is retrieved for an existing detector with no FLPs assigned", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/inventory/detectors/MID/flps/json", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should return an empty list", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))

						var hosts []string
						err = json.NewDecoder(recorder.Body).Decode(&hosts)
						Expect(hosts).To(BeEmpty())
					})
				})
			})
			Context("expecting plain text output", func() {
				When("the list of hosts is retrieved for all detectors", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/inventory/flps", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should contain all the hosts for all the detectors listed in the test YAML file at o2/hardware/detectors", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))
						// order is not guaranteed
						Expect(recorder.Body.String()).To(ContainSubstring("flp100"))
						Expect(recorder.Body.String()).To(ContainSubstring("flp001"))
						Expect(recorder.Body.String()).To(ContainSubstring("flp002"))
						Expect(recorder.Body.String()).To(ContainSubstring("flp003"))
					})
				})
				When("the list of hosts is retrieved for a valid detector", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/inventory/detectors/ITS/flps", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should contain all the hosts for the ITS detector in the test YAML file at o2/hardware/detectors", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))
						Expect(recorder.Body.String()).To(Equal("flp001\n"))
					})
				})
			})
		})
		Describe("getting the detectors FLP inventory", func() {
			Context("expecting JSON output", func() {
				var inventory map[string][]string
				When("the inventory is retrieved", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/inventory/detectors/json", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should contain all the hosts for all the detectors in the test YAML file at o2/hardware/detectors", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))

						err = json.NewDecoder(recorder.Body).Decode(&inventory)

						Expect(err).NotTo(HaveOccurred())
						Expect(inventory).To(HaveKey("ITS"))
						Expect(inventory["ITS"]).To(ContainElements("flp001"))
						Expect(inventory).To(HaveKey("HMP"))
						Expect(inventory["HMP"]).To(ContainElements("flp002", "flp003"))
						Expect(inventory).To(HaveKey("TRG"))
						Expect(inventory["TRG"]).To(ContainElements("flp100"))
						Expect(inventory).To(HaveKey("MID"))
						Expect(inventory["MID"]).To(BeEmpty())
					})
				})
			})
			Context("expecting plain text output", func() {
				When("the inventory is retrieved", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/inventory/detectors/text", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should contain all the hosts for all the detectors in the test YAML file at o2/hardware/detectors", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))
						Expect(recorder.Body.String()).To(ContainSubstring("ITS\n\tflp001"))
						Expect(recorder.Body.String()).To(ContainSubstring("TRG\n\tflp100"))
						Expect(recorder.Body.String()).To(ContainSubstring("MID\n"))
						Expect(recorder.Body.String()).To(ContainSubstring("HMP\n")) // flps can be in any order, we won't test it
					})
				})
			})
		})

		Describe("getting component configuration", func() {
			// fixme: compared to the raw local/service, the HTTP service adds a newline character.
			//  probably not an issue, but still an inconsistency to be aware of.
			When("requesting an entry for a concrete run type and a concrete role", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/PHYSICS/role1/entry1", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should return the payload for the concrete run type and role", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("entry1 config PHYSICS role1\n"))
				})
			})
			When("requesting an entry for ANY run type and any role", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/ANY/any/entry1", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should return the payload for ANY/any", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("entry1 config ANY any\n"))
				})
			})
			When("requesting an entry for ANY run type and any role with '/' at the end of the query", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/ANY/any/entry1/", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should return the payload for ANY/any", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("entry1 config ANY any\n"))
				})
			})
			When("requesting an entry in a subfolder", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/ANY/any/sub/entry12", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should return the expected payload", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("world\n"))
				})
			})
		})

		Describe("getting and processing component configuration", func() {
			When("requesting an entry which requires including another entry and inserting a variable", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/ANY/any/entry10?process=true&var1=hello", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should return the processed payload", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("hello world\n"))
				})
			})
			When("requesting an entry which requires including another entry in a different node", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/ANY/any/entry12?process=true", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should return the processed payload", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("hello world\n"))
				})
			})
			When("requesting an entry which is in a subfolder", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/ANY/any/sub/entry12?process=true", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should return the processed payload", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("world\n"))
				})
			})
		})
		Describe("resolving a query", func() {
			// fixme: compared to the raw local/service, the HTTP service adds a newline character.
			//  probably not an issue, but still an inconsistency to be aware of.
			When("resolving an ANY/any query", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/ANY/any/entry1/resolve", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should resolve to ANY/any query", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("qc/ANY/any/entry1\n"))
				})
			})
			When("resolving a concrete PHYSICS/role1 query while there is ANY/any entry in the configuration tree", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/PHYSICS/role1/entry11/resolve", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should resolve to ANY/any query", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("qc/ANY/any/entry11\n"))
				})
			})
			When("resolving a query to an entry in a subfolder", func() {
				BeforeEach(func() {
					req, err := http.NewRequest("GET", "/components/qc/ANY/any/sub/entry12/resolve", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
				})
				It("should resolve it correctly", func() {
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("qc/ANY/any/sub/entry12\n"))
				})
			})
		})

		Describe("listing all entries for a component/runtype/rolename", func() {
			Context("expecting plain text output", func() {
				When("listing all entries (without '/' at the end)", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/components/qc/ANY/any", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should return text list with the entries, one per line", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))
						Expect(recorder.Body.String()).To(Equal(
							"ANY/any/entry1\n" +
								"ANY/any/entry10\n" +
								"ANY/any/entry11\n" +
								"ANY/any/entry12\n" +
								"ANY/any/sub/entry12\n"))
					})
				})
				When("listing all entries (with '/' at the end)", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/components/qc/ANY/any/", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should return text list with the entries, one per line", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))
						Expect(recorder.Body.String()).To(Equal(
							"ANY/any/entry1\n" +
								"ANY/any/entry10\n" +
								"ANY/any/entry11\n" +
								"ANY/any/entry12\n" +
								"ANY/any/sub/entry12\n"))
					})
				})
			})
			Context("expecting json output", func() {
				When("listing all entries", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/components/qc/ANY/any?format=json", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should return text list with the entries, one per line", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))
						var hosts []string
						err = json.NewDecoder(recorder.Body).Decode(&hosts)
						Expect(hosts).To(ContainElements("ANY/any/entry1", "ANY/any/entry10", "ANY/any/entry11", "ANY/any/entry12", "ANY/any/sub/entry12"))
					})
				})
			})
		})

		Describe("listing components", func() {
			Context("with text/plain format", func() {
				When("the list of components is retrieved", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/components?format=text/plain", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should have the expected components", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))
						Expect(recorder.Body.String()).To(Equal("qc\n"))
					})
				})
			})
			Context("with json format", func() {
				When("the list of components is retrieved", func() {
					BeforeEach(func() {
						req, err := http.NewRequest("GET", "/components?format=json", nil)
						Expect(err).NotTo(HaveOccurred())
						handler.ServeHTTP(recorder, req)
					})
					It("should have the expected components", func() {
						Expect(recorder.Code).To(Equal(http.StatusOK))

						var components []string
						err = json.NewDecoder(recorder.Body).Decode(&components)
						Expect(err).NotTo(HaveOccurred())
						Expect(components).To(HaveLen(1))
						Expect(components).To(ContainElements("qc"))
					})
				})
			})
		})

		Describe("invalidating template cache", func() {
			When("requesting an entry after having invalidated cache", func() {
				It("should provide a valid entry", func() {
					req, err := http.NewRequest("GET", "/components/qc/PHYSICS/role1/entry1", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("entry1 config PHYSICS role1\n"))

					recorder = httptest.NewRecorder()
					req, err = http.NewRequest("POST", "/components/_invalidate_cache", nil)
					handler.ServeHTTP(recorder, req)
					Expect(recorder.Code).To(Equal(http.StatusOK))

					recorder = httptest.NewRecorder()
					req, err = http.NewRequest("GET", "/components/qc/PHYSICS/role1/entry1", nil)
					Expect(err).NotTo(HaveOccurred())
					handler.ServeHTTP(recorder, req)
					Expect(recorder.Code).To(Equal(http.StatusOK))
					Expect(recorder.Body.String()).To(Equal("entry1 config PHYSICS role1\n"))
				})
			})
		})
	})
})
