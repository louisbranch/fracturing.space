package openai

import (
	"encoding/json"
	"strings"
)

func openAIToolSchema(schema any) map[string]any {
	value := cloneSchemaMap(schema)
	if value == nil {
		return map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		}
	}
	enforceStrictSchema(value)
	return value
}

// enforceStrictSchema recursively ensures every object node has
// additionalProperties: false and a required array listing all properties,
// as mandated by OpenAI strict mode.
func enforceStrictSchema(node map[string]any) {
	if strings.TrimSpace(stringValue(node["type"])) == "" {
		node["type"] = "object"
	}
	if strings.EqualFold(stringValue(node["type"]), "object") {
		props, ok := node["properties"].(map[string]any)
		if !ok || props == nil {
			node["properties"] = map[string]any{}
			props = node["properties"].(map[string]any)
		}
		if _, ok := node["additionalProperties"]; !ok {
			node["additionalProperties"] = false
		}
		if _, ok := node["required"]; !ok {
			required := make([]string, 0, len(props))
			for key := range props {
				required = append(required, key)
			}
			if len(required) > 0 {
				node["required"] = required
			}
		}
		for _, propValue := range props {
			if propMap, ok := propValue.(map[string]any); ok {
				enforceStrictSchema(propMap)
			}
		}
	}
	if strings.EqualFold(stringValue(node["type"]), "array") {
		if items, ok := node["items"].(map[string]any); ok {
			enforceStrictSchema(items)
		}
	}
}

func cloneSchemaMap(schema any) map[string]any {
	if schema == nil {
		return nil
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil
	}
	return value
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
