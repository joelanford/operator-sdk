package crd_test

import (
	"bytes"
	"text/template"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/internal/templates/config/crd"
)

var _ = Describe("Crd", func() {
	var f crd.CRD
	BeforeEach(func() {
		c := config.Config{Domain: "example.com"}
		opts := resource.Options{
			Group:      "test",
			Version:    "v1alpha1",
			Kind:       "Foo",
			Plural:     "foos",
			Namespaced: true,
		}
		f = crd.CRD{}
		f.Resource = opts.NewResource(&c, false)
	})

	Describe("SetTemplateDefaults", func() {
		When("CRDVersion is invalid", func() {
			BeforeEach(func() { f.CRDVersion = "invalid" })
			It("should fail", func() {
				Expect(f.SetTemplateDefaults()).NotTo(Succeed())
			})
		})
		When("CRDVersion is empty", func() {
			It("should scaffold a v1 CRD", func() {
				data := execTemplate(f)
				v1 := apiextv1.CustomResourceDefinition{}
				Expect(yaml.Unmarshal(data, &v1)).To(Succeed())
				Expect(v1.APIVersion).To(Equal(apiextv1.SchemeGroupVersion.String()))
			})
		})
		When("CRDVersion is v1", func() {
			BeforeEach(func() { f.CRDVersion = "v1" })
			It("should scaffold a v1 CRD", func() {
				data := execTemplate(f)
				v1 := apiextv1.CustomResourceDefinition{}
				Expect(yaml.Unmarshal(data, &v1)).To(Succeed())
				Expect(v1.APIVersion).To(Equal(apiextv1.SchemeGroupVersion.String()))
			})
		})
		When("CRDVersion is v1beta1", func() {
			BeforeEach(func() { f.CRDVersion = "v1beta1" })
			It("should scaffold a v1beta1 CRD", func() {
				data := execTemplate(f)
				v1beta1 := apiextv1beta1.CustomResourceDefinition{}
				Expect(yaml.Unmarshal(data, &v1beta1)).To(Succeed())
				Expect(v1beta1.APIVersion).To(Equal(apiextv1beta1.SchemeGroupVersion.String()))
			})
		})
		When("SpecSchema is empty", func() {
			It("should scaffold a default spec schema", func() {
				data := execTemplate(f)
				v1 := apiextv1.CustomResourceDefinition{}
				expectedXPreserveUnknownFields := true
				Expect(yaml.Unmarshal(data, &v1)).To(Succeed())
				Expect(v1.Spec.Versions).To(HaveLen(1))
				Expect(v1.Spec.Versions[0].Schema).NotTo(BeNil())
				Expect(v1.Spec.Versions[0].Schema.OpenAPIV3Schema).NotTo(BeNil())
				Expect(v1.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]).To(Equal(
					apiextv1.JSONSchemaProps{
						Description:            "Spec defines the desired state of Foo",
						Type:                   "object",
						XPreserveUnknownFields: &expectedXPreserveUnknownFields,
					},
				))
			})
		})
		When("SpecSchema is not empty", func() {
			var (
				baseSchema apiextv1.JSONSchemaProps
			)

			BeforeEach(func() {
				baseSchema = apiextv1.JSONSchemaProps{
					Type: "object",
					Properties: map[string]apiextv1.JSONSchemaProps{
						"test": {Type: "string"},
					},
				}
				f.SpecSchema = &baseSchema
			})
			When("Description is empty", func() {
				BeforeEach(func() { f.SpecSchema.Description = "" })
				It("should scaffold a default spec schema", func() {
					data := execTemplate(f)
					v1 := apiextv1.CustomResourceDefinition{}
					Expect(yaml.Unmarshal(data, &v1)).To(Succeed())
					testSpecSchema(v1, withDesc(baseSchema, "Spec defines the desired state of Foo"))
				})
			})
			When("Description is not empty", func() {
				const customDescription = "Custom description"
				BeforeEach(func() { f.SpecSchema.Description = customDescription })
				It("should scaffold a default spec schema", func() {
					data := execTemplate(f)
					v1 := apiextv1.CustomResourceDefinition{}
					Expect(yaml.Unmarshal(data, &v1)).To(Succeed())
					testSpecSchema(v1, withDesc(baseSchema, customDescription))
				})
			})
		})
	})
})

func execTemplate(f crd.CRD) []byte {
	Expect(f.SetTemplateDefaults()).To(Succeed())
	t, err := template.New("crd").Funcs(f.GetFuncMap()).Parse(f.GetBody())
	Expect(err).To(BeNil())

	var buf bytes.Buffer
	Expect(t.Execute(&buf, f)).To(Succeed())
	return buf.Bytes()
}

func withDesc(s apiextv1.JSONSchemaProps, desc string) apiextv1.JSONSchemaProps {
	s.Description = desc
	return s
}

func testSpecSchema(v1 apiextv1.CustomResourceDefinition, expSchema apiextv1.JSONSchemaProps) {
	Expect(v1.Spec.Versions).To(HaveLen(1))
	Expect(v1.Spec.Versions[0].Schema).NotTo(BeNil())
	Expect(v1.Spec.Versions[0].Schema.OpenAPIV3Schema).NotTo(BeNil())
	Expect(v1.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]).To(Equal(expSchema))
}
