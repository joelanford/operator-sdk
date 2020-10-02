package scaffolds

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/chart"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var _ = Describe("Api", func() {
	Describe("loadSchema", func() {
		var chrt, dep *chart.Chart
		BeforeEach(func() {
			chrt = &chart.Chart{Metadata: &chart.Metadata{Name: "test"}}
			dep = &chart.Chart{Metadata: &chart.Metadata{Name: "dep"}}
		})
		When("chart has no schema", func() {
			When("chart has no dependencies", func() {
				It("should return a nil schema", func() {
					s, err := loadSchema(chrt)
					Expect(err).To(BeNil())
					Expect(s).To(BeNil())
				})
			})
			When("chart has dependencies", func() {
				BeforeEach(func() { chrt.AddDependency(dep) })
				When("none have schemas", func() {
					It("should return a nil schema", func() {
						s, err := loadSchema(chrt)
						Expect(err).To(BeNil())
						Expect(s).To(BeNil())
					})
				})
				When("at least one has a schema", func() {
					BeforeEach(func() {
						dep.Schema = []byte(`{"type":"object", "properties":{"depField":{"type":"string"}}}`)
					})
					It("should return a schema containing the nested dependency's schema as a property keyed by the dependency's name", func() {
						s, err := loadSchema(chrt)
						Expect(err).To(BeNil())
						Expect(s).To(Equal(&apiextv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextv1.JSONSchemaProps{
								"dep": {
									Type: "object",
									Properties: map[string]apiextv1.JSONSchemaProps{
										"depField": {Type: "string"},
									},
								},
							},
						}))
					})
				})
			})
		})
		When("chart has a schema", func() {
			BeforeEach(func() {
				chrt.Schema = []byte(`{"type":"object", "properties":{"rootField":{"type":"string"}}}`)
			})
			When("chart has no dependencies", func() {
				It("should return a schema from the chart's schema", func() {
					s, err := loadSchema(chrt)
					Expect(err).To(BeNil())
					Expect(s).To(Equal(&apiextv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextv1.JSONSchemaProps{
							"rootField": {Type: "string"},
						},
					}))
				})
			})
			When("chart has dependencies", func() {
				BeforeEach(func() { chrt.AddDependency(dep) })
				When("none have schemas", func() {
					It("should return a schema from the chart's schema", func() {
						s, err := loadSchema(chrt)
						Expect(err).To(BeNil())
						Expect(s).To(Equal(&apiextv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextv1.JSONSchemaProps{
								"rootField": {Type: "string"},
							},
						}))
					})
				})
				When("at least one has a schema", func() {
					BeforeEach(func() {
						dep.Schema = []byte(`{"type":"object", "properties":{"depField":{"type":"string"}}}`)
					})
					It("should return a schema from the chart's schema", func() {
						s, err := loadSchema(chrt)
						Expect(err).To(BeNil())
						Expect(s).To(Equal(&apiextv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextv1.JSONSchemaProps{
								"rootField": {Type: "string"},
								"dep": {
									Type: "object",
									Properties: map[string]apiextv1.JSONSchemaProps{
										"depField": {Type: "string"},
									},
								},
							},
						}))
					})
				})
			})
		})
	})
})
