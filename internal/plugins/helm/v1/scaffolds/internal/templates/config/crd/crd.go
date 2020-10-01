// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
	"sigs.k8s.io/yaml"
)

var _ file.Template = &CRD{}

// CRD scaffolds a manifest for CRD sample.
type CRD struct {
	file.TemplateMixin
	file.ResourceMixin

	CRDVersion string
	SpecSchema *apiextv1.JSONSchemaProps
}

// SetTemplateDefaults implements input.Template
func (f *CRD) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "bases", fmt.Sprintf("%s_%%[plural].yaml", f.Resource.Domain))
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	f.IfExistsAction = file.Error

	specDescription := fmt.Sprintf("Spec defines the desired state of %s", f.Resource.Kind)
	if f.SpecSchema == nil {
		preserveUnknownFields := true
		f.SpecSchema = &apiextv1.JSONSchemaProps{
			Description:            specDescription,
			Type:                   "object",
			XPreserveUnknownFields: &preserveUnknownFields,
		}
	} else if f.SpecSchema.Description == "" {
		f.SpecSchema.Description = specDescription
	}

	switch f.CRDVersion {
	case "", "v1":
		f.TemplateBody = crdTemplateV1
	case "v1beta1":
		f.TemplateBody = crdTemplateV1Beta1
	default:
		return errors.New("the CRD version value must be either 'v1' or 'v1beta1'")
	}
	return nil
}

func (f *CRD) GetFuncMap() template.FuncMap {
	fm := file.DefaultFuncMap()
	fm["toYaml"] = func(v interface{}) string {
		out, _ := yaml.Marshal(v)
		return string(out)
	}
	fm["indent"] = func(spaces int, v string) string {
		pad := strings.Repeat(" ", spaces)
		return pad + strings.Replace(v, "\n", "\n"+pad, -1)
	}
	fm["trim"] = strings.TrimSpace
	return fm
}

const crdTemplateV1 = `---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: {{ .Resource.Plural }}.{{ .Resource.Domain }}
spec:
  group: {{ .Resource.Domain }}
  names:
    kind: {{ .Resource.Kind }}
    listKind: {{ .Resource.Kind }}List
    plural: {{ .Resource.Plural }}
    singular: {{ .Resource.Kind | lower }}
  scope: Namespaced
  versions:
  - name: {{ .Resource.Version }}
    schema:
      openAPIV3Schema:
        description: {{ .Resource.Kind }} is the Schema for the {{ .Resource.Plural }} API
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
{{ .SpecSchema | toYaml | trim | indent 12 }}
          status:
            description: Status defines the observed state of {{ .Resource.Kind }}
            type: object
            x-kubernetes-preserve-unknown-fields: true
        type: object
    served: true
    storage: true
    subresources:
      status: {}
`

const crdTemplateV1Beta1 = `---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{ .Resource.Plural }}.{{ .Resource.Domain }}
spec:
  group: {{ .Resource.Domain }}
  names:
    kind: {{ .Resource.Kind }}
    listKind: {{ .Resource.Kind }}List
    plural: {{ .Resource.Plural }}
    singular: {{ .Resource.Kind | lower }}
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      description: {{ .Resource.Kind }} is the Schema for the {{ .Resource.Plural }} API
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
{{ .SpecSchema | toYaml | trim | indent 10 }}
        status:
          description: Status defines the observed state of {{ .Resource.Kind }}
          type: object
          x-kubernetes-preserve-unknown-fields: true
      type: object
    versions:
    - name: {{ .Resource.Version }}
      served: true
      storage: true
`
