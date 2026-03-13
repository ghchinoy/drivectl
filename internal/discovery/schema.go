package discovery

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
)

// MethodToSchema converts a discovery RestMethod and its associated parameters into an MCP jsonschema.Schema.
func MethodToSchema(method *RestMethod, doc *RestDescription) *jsonschema.Schema {
	properties := make(map[string]*jsonschema.Schema)
	var required []string

	for name, param := range method.Parameters {
		paramSchema := &jsonschema.Schema{
			Type:        param.Type,
			Description: param.Description,
		}
		if param.Default != "" {
			b, _ := json.Marshal(param.Default)
			paramSchema.Default = json.RawMessage(b)
		}
		if len(param.Enum) > 0 {
			var enums []any
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

	if method.Request != nil && method.Request.Ref != "" {
		if reqSchema, ok := doc.Schemas[method.Request.Ref]; ok {
			payloadSchema := convertJsonSchema(&reqSchema, doc)
			properties["payload"] = payloadSchema
			required = append(required, "payload")
		}
	}

	return &jsonschema.Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

func convertJsonSchema(s *JsonSchema, doc *RestDescription) *jsonschema.Schema {
	if s == nil {
		return nil
	}
	schemaType := s.Type
	if schemaType == "" && s.Ref != "" {
		if resolved, ok := doc.Schemas[s.Ref]; ok {
			return convertJsonSchema(&resolved, doc)
		}
		schemaType = "object"
	}

	props := make(map[string]*jsonschema.Schema)
	for k, v := range s.Properties {
		propType := v.Type
		if propType == "" && v.Ref != "" {
			if resolved, ok := doc.Schemas[v.Ref]; ok {
				props[k] = convertJsonSchema(&resolved, doc)
				continue
			}
		}
		
		var items *jsonschema.Schema
		if v.Items != nil {
			itemType := v.Items.Type
			if itemType == "" && v.Items.Ref != "" {
				if resolved, ok := doc.Schemas[v.Items.Ref]; ok {
					items = convertJsonSchema(&resolved, doc)
				}
			} else {
				items = &jsonschema.Schema{Type: itemType}
			}
		}

		props[k] = &jsonschema.Schema{
			Type:        propType,
			Description: v.Description,
		}
		if items != nil {
			props[k].Items = items
		}
	}

	return &jsonschema.Schema{
		Type:        schemaType,
		Description: s.Description,
		Properties:  props,
		Required:    s.Required,
	}
}
