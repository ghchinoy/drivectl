// Copyright 2025 Google LLC
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

package discovery

import (
	"encoding/json"
	"maps"
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
)

// MethodToSchema converts a Discovery RestMethod and its associated parameters into a JSON Schema.
func MethodToSchema(method *RestMethod, doc *RestDescription) *jsonschema.Schema {
	properties := make(map[string]*jsonschema.Schema)
	var required []string

	for name, param := range method.Parameters {
		paramSchema := &jsonschema.Schema{
			Type:        normalizeSchemaType(param.Type),
			Description: param.Description,
		}
		if rawDefault := encodeParamDefault(param.Type, param.Default); rawDefault != nil {
			paramSchema.Default = rawDefault
		}
		if len(param.Enum) > 0 {
			enums := make([]any, 0, len(param.Enum))
			for _, v := range param.Enum {
				enums = append(enums, v)
			}
			paramSchema.Enum = enums
		}
		properties[name] = paramSchema
		if param.Required {
			required = append(required, name)
		}
	}

	if method.Request != nil && method.Request.Ref != "" && doc != nil {
		if reqSchema, ok := doc.Schemas[method.Request.Ref]; ok {
			visited := map[string]bool{method.Request.Ref: true}
			properties["payload"] = convertJSONSchema(&reqSchema, doc, visited)
			required = append(required, "payload")
		}
	}

	return &jsonschema.Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

func normalizeSchemaType(schemaType string) string {
	if schemaType == "any" {
		return ""
	}
	return schemaType
}

func encodeParamDefault(paramType, defaultVal string) json.RawMessage {
	if defaultVal == "" {
		return nil
	}
	switch paramType {
	case "boolean":
		if b, err := strconv.ParseBool(defaultVal); err == nil {
			raw, _ := json.Marshal(b)
			return json.RawMessage(raw)
		}
	case "integer":
		if n, err := strconv.ParseInt(defaultVal, 10, 64); err == nil {
			raw, _ := json.Marshal(n)
			return json.RawMessage(raw)
		}
	case "number":
		if f, err := strconv.ParseFloat(defaultVal, 64); err == nil {
			raw, _ := json.Marshal(f)
			return json.RawMessage(raw)
		}
	}
	raw, err := json.Marshal(defaultVal)
	if err != nil {
		return nil
	}
	return json.RawMessage(raw)
}

func convertJSONSchema(s *JSONSchema, doc *RestDescription, visited map[string]bool) *jsonschema.Schema {
	if s == nil {
		return nil
	}
	schemaType := normalizeSchemaType(s.Type)
	if schemaType == "" && s.Ref != "" {
		return resolveSchemaRef(s.Ref, s.Description, doc, visited)
	}

	props := make(map[string]*jsonschema.Schema, len(s.Properties))
	for k, v := range s.Properties {
		props[k] = convertSchemaProperty(v, doc, visited)
	}

	return &jsonschema.Schema{
		Type:        schemaType,
		Description: s.Description,
		Properties:  props,
		Required:    s.Required,
	}
}

func convertSchemaProperty(v JSONSchemaProperty, doc *RestDescription, visited map[string]bool) *jsonschema.Schema {
	propType := normalizeSchemaType(v.Type)
	if propType == "" && v.Ref != "" {
		return resolveSchemaRef(v.Ref, v.Description, doc, visited)
	}

	propSchema := &jsonschema.Schema{
		Type:        propType,
		Description: v.Description,
	}
	if v.Items != nil {
		propSchema.Items = convertSchemaItems(v.Items, doc, visited)
	}
	return propSchema
}

func convertSchemaItems(items *JSONSchemaProperty, doc *RestDescription, visited map[string]bool) *jsonschema.Schema {
	itemType := normalizeSchemaType(items.Type)
	if itemType == "" && items.Ref != "" {
		return resolveSchemaRef(items.Ref, items.Description, doc, visited)
	}
	return &jsonschema.Schema{
		Type:        itemType,
		Description: items.Description,
	}
}

func resolveSchemaRef(ref, fallbackDesc string, doc *RestDescription, visited map[string]bool) *jsonschema.Schema {
	if doc == nil {
		return &jsonschema.Schema{Type: "object", Description: fallbackDesc}
	}
	resolved, ok := doc.Schemas[ref]
	if !ok {
		return &jsonschema.Schema{Type: "object", Description: fallbackDesc}
	}
	if visited[ref] {
		desc := resolved.Description
		if desc == "" {
			desc = fallbackDesc
		}
		return &jsonschema.Schema{Type: "object", Description: desc}
	}
	nextVisited := maps.Clone(visited)
	if nextVisited == nil {
		nextVisited = make(map[string]bool)
	}
	nextVisited[ref] = true
	return convertJSONSchema(&resolved, doc, nextVisited)
}
