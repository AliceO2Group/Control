package workflow

import (
	"github.com/AliceO2Group/Control/core/repos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("aggregator role", func() {
	var _ = Describe("processing templates", func() {
		var root Role
		var repo repos.Repo
		var configStack map[string]string

		BeforeEach(func() {
			_, repo, _ = repos.NewRepo("/home/user/git/ControlWorkflows", "", "/var/lib/o2/aliecs/repos")
			configStack = make(map[string]string)
		})

		When("an aggregator role is empty", func() {
			BeforeEach(func() {
				root = &aggregatorRole{
					roleBase{Name: "root", Enabled: "true"},
					aggregator{Roles: []Role{}},
				}
			})
			It("should disable itself", func() {
				Expect(root.IsEnabled()).To(BeTrue())
				err := root.ProcessTemplates(&repo, nil, configStack)
				Expect(err).NotTo(HaveOccurred())
				Expect(root.IsEnabled()).To(BeFalse())
			})
		})

		When("an aggregator role has only disabled sub-roles", func() {
			BeforeEach(func() {
				root = &aggregatorRole{
					roleBase{Name: "root", Enabled: "true"},
					aggregator{
						Roles: []Role{&taskRole{roleBase: roleBase{Name: "task1", Enabled: "false"}}},
					},
				}
			})
			It("remove the disabled roles and disable itself", func() {
				Expect(root.IsEnabled()).To(BeTrue())
				err := root.ProcessTemplates(&repo, nil, configStack)
				Expect(err).NotTo(HaveOccurred())
				Expect(root.IsEnabled()).To(BeFalse())
				Expect(root.GetRoles()).Should(BeEmpty())
			})
		})

		When("an aggregator role has an enabled role", func() {
			BeforeEach(func() {
				root = &aggregatorRole{
					roleBase{Name: "root", Enabled: "true"},
					aggregator{
						Roles: []Role{&taskRole{roleBase: roleBase{Name: "task1", Enabled: "true"}}},
					},
				}
			})
			It("should be enabled", func() {
				Expect(root.IsEnabled()).To(BeTrue())
				err := root.ProcessTemplates(&repo, nil, configStack)
				Expect(err).NotTo(HaveOccurred())
				Expect(root.IsEnabled()).To(BeTrue())
				Expect(root.GetRoles()).ShouldNot(BeEmpty())
			})
		})
	})

})
