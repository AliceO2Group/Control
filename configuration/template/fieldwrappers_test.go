package template_test

import (
	"github.com/AliceO2Group/Control/configuration/template"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Field Wrappers", func() {
	Describe("PointerWrapper", func() {
		It("should get and set values correctly", func() {
			value := "test"
			pw := template.WrapPointer(&value)

			Expect(pw.Get()).To(Equal("test"))

			pw.Set("new value")
			Expect(value).To(Equal("new value"))
		})
	})

	Describe("GenericWrapper", func() {
		It("should get and set values correctly", func() {
			value := "test"
			gw := template.WrapGeneric(
				func() string { return value },
				func(v string) { value = v },
			)

			Expect(gw.Get()).To(Equal("test"))

			gw.Set("new value")
			Expect(value).To(Equal("new value"))
		})
	})

	Describe("WrapMapItems", func() {
		It("should wrap map items correctly", func() {
			items := map[string]string{
				"key1": "value1",
				"key2": "value2",
			}
			fields := template.WrapMapItems(items)

			Expect(fields).To(HaveLen(2))

			for _, field := range fields {
				initialValue := field.Get()
				field.Set("new " + initialValue)
			}

			expectedItems := map[string]string{
				"key1": "new value1",
				"key2": "new value2",
			}

			Expect(items).To(Equal(expectedItems))
		})
	})

	Describe("WrapSliceItems", func() {
		It("should wrap slice items correctly", func() {
			items := []string{"item1", "item2", "item3"}
			fields := template.WrapSliceItems(items)

			Expect(fields).To(HaveLen(3))

			for i, field := range fields {
				Expect(field.Get()).To(Equal(items[i]))
				field.Set("new " + items[i])
			}

			expectedItems := []string{"new item1", "new item2", "new item3"}

			Expect(items).To(Equal(expectedItems))
		})
	})
})
