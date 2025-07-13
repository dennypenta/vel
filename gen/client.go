package gen

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"unicode"

	"github.com/dennypenta/vel"
	"gopkg.in/yaml.v3"
)

var ErrorInlineStructForbidden = errors.New("inlined structs are forbidden to use, declare an explicit type")

// ClientGen defines api client generator
// it doesn't support the following:
// - anonymous nested struct
type ClientGen struct {
	meta ApiClientDesc
}

func New(clientDesc ClientDesc, meta []vel.HandlerMeta) (*ClientGen, error) {
	// Pre-calculate client description values
	clientDesc.TypeNameLower = strings.ToLower(clientDesc.TypeName)

	desc := make([]ApiDesc, len(meta))
	dataTypeSet := make(map[string]struct{}, len(meta)*2)

	var err error
	for i := range meta {
		dataTypes := make([]DataType, 0, len(meta)*2)
		desc[i], err = makeApiDesc(meta[i])
		if err != nil {
			return nil, err
		}

		name := desc[i].Input.Name
		if _, ok := dataTypeSet[name]; !ok && len(desc[i].Input.Fields) > 0 {
			dataTypes = append(dataTypes, desc[i].Input)
			dataTypeSet[name] = struct{}{}
		}
		name = desc[i].Output.Name
		if _, ok := dataTypeSet[name]; !ok && len(desc[i].Output.Fields) > 0 {
			dataTypes = append(dataTypes, desc[i].Output)
			dataTypeSet[name] = struct{}{}
		}

		for j := range desc[i].Input.Fields {
			types, err := collectStructs(desc[i].Input.Fields[j], dataTypeSet)
			if err != nil {
				return nil, err
			}
			dataTypes = append(dataTypes, types...)
		}
		for j := range desc[i].Output.Fields {
			types, err := collectStructs(desc[i].Output.Fields[j], dataTypeSet)
			if err != nil {
				return nil, err
			}
			dataTypes = append(dataTypes, types...)
		}

		desc[i].DataTypes = dataTypes
	}
	return &ClientGen{
		meta: ApiClientDesc{
			Client: clientDesc,
			Apis:   desc,
		},
	}, nil
}

func collectStructs(field Field, dataTypeSet map[string]struct{}) ([]DataType, error) {
	dataTypes := make([]DataType, 0)

	if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Map {
		field.Type = field.Type.Elem()
	}
	if field.Type.Kind() == reflect.Pointer {
		field.Type = field.Type.Elem()
	}
	if field.Type.Kind() == reflect.Struct {
		subTypes, err := collectTypes(field, dataTypeSet)
		if err != nil {
			return nil, err
		}
		dataTypes = append(dataTypes, subTypes...)
	}

	return dataTypes, nil
}

func collectTypes(field Field, dataTypeSet map[string]struct{}) ([]DataType, error) {
	if _, ok := builtinTypes[field.TypeName]; ok {
		return nil, nil
	}
	dataTypes := make([]DataType, 0)
	subType, err := extractDataType(field.Type)
	if err != nil {
		return nil, err
	}
	name := subType.Name
	if _, ok := dataTypeSet[name]; !ok && len(subType.Fields) > 0 {
		dataTypes = append(dataTypes, subType)
		dataTypeSet[name] = struct{}{}

		for _, subField := range subType.Fields {
			// don't need to generate this type.
			if subField.IsBuilting {
				continue
			}
			if subField.Type.Kind() == reflect.Struct {
				subTypes, err := collectTypes(subField, dataTypeSet)
				if err != nil {
					return nil, err
				}

				dataTypes = append(dataTypes, subTypes...)
			}
			if subField.Type.Kind() == reflect.Slice || subField.Type.Kind() == reflect.Map || subField.Type.Kind() == reflect.Pointer {
				if subField.Type.Elem().Kind() == reflect.Pointer {
					subField.Type = subField.Type.Elem()
				}
				if subField.Type.Elem().Kind() == reflect.Struct {
					subField.Type = subField.Type.Elem()
					subTypes, err := collectTypes(subField, dataTypeSet)
					if err != nil {
						return nil, err
					}

					dataTypes = append(dataTypes, subTypes...)
				}
			}
		}
	}

	return dataTypes, nil
}

func makeApiDesc(meta vel.HandlerMeta) (ApiDesc, error) {
	inputReflectType := reflect.TypeOf(meta.Input)
	inputType, err := extractDataType(inputReflectType)
	if err != nil {
		return ApiDesc{}, err
	}
	outputReflectType := reflect.TypeOf(meta.Output)
	outputType, err := extractDataType(outputReflectType)
	if err != nil {
		return ApiDesc{}, err
	}

	return ApiDesc{
		Input:       inputType,
		Output:      outputType,
		OperationID: meta.OperationID,
		Method:      meta.Method,
		FuncName:    Capitalize(meta.OperationID),
		Spec:        meta.Spec,
	}, nil
}

func extractDataType(t reflect.Type) (DataType, error) {
	var fields []Field

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		typeName := field.Type.String()
		isBuiltin := false

		if _, ok := builtinTypes[typeName]; ok {
			isBuiltin = true
		} else {
			if field.Type.Kind() == reflect.Struct {
				typeName = field.Type.Name()
			}
			if field.Type.Kind() == reflect.Pointer && field.Type.Elem().Kind() == reflect.Struct {
				typeName = "*" + field.Type.Elem().Name()
			}
			if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct {
				typeName = "[]" + field.Type.Elem().Name()
			}
			if field.Type.Kind() == reflect.Map && field.Type.Elem().Kind() == reflect.Struct {
				typeName = "map[" + field.Type.Key().Name() + "]" + field.Type.Elem().Name()
			}
			if field.Type.Kind() == reflect.String && typeName != reflect.String.String() {
				typeName = reflect.String.String()
			}
		}

		fields = append(fields, Field{
			Name:       field.Name,
			Type:       field.Type,
			TypeName:   typeName,
			TSTypeName: toTSType(typeName),
			JsonTag:    field.Tag.Get("json"),
			SchemaTag:  field.Tag.Get("schema"),
			IsBuilting: isBuiltin,
		})
	}

	name := t.Name()
	if len(fields) == 0 {
		name = ""
	}

	if name == "" && len(fields) > 0 {
		return DataType{}, ErrorInlineStructForbidden
	}
	return DataType{
		Name:   name,
		Fields: fields,
	}, nil
}

type ApiClientDesc struct {
	Client ClientDesc
	Apis   []ApiDesc
}

type ClientDesc struct {
	TypeName      string
	PackageName   string
	TypeNameLower string
}

type ApiDesc struct {
	Input       DataType
	Output      DataType
	OperationID string
	Method      string
	FuncName    string
	DataTypes   []DataType
	Spec        vel.Spec
}

type DataType struct {
	Name   string
	Fields []Field
	// OtherTypes defines a list of types required to generate the fields
	OtherTypes []DataType
}

type Field struct {
	Name       string
	Type       reflect.Type
	TypeName   string
	TSTypeName string // TypeScript type name
	JsonTag    string
	SchemaTag  string
	// IsBuiltin defines a flag that a field exists in std lib, therefore must not be broken down further
	// e.g. time.Time
	IsBuilting bool
}

func (g *ClientGen) Generate(w io.Writer, templateName, postProcessing string) error {
	pipe := bytes.NewBuffer(nil)
	clientTpl, ok := templateRegistry[templateName]
	if !ok {
		return fmt.Errorf("template %s not found", templateName)
	}

	if err := clientTpl.Execute(pipe, g.meta); err != nil {
		return err
	}

	bytes := pipe.Bytes()
	if postProcessing != "" {
		f, err := os.CreateTemp("", "")
		if err != nil {
			return err
		}
		if _, err := f.Write(bytes); err != nil {
			return err
		}
		defer f.Close()
		cmdStr := fmt.Sprintf("%s < %s", postProcessing, f.Name())

		bytes, err = exec.Command("sh", "-c", cmdStr).Output()
		if err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}

	}

	if _, err := w.Write(bytes); err != nil {
		return err
	}

	return nil
}

func Capitalize(s string) string {
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	s = string(r)
	return s
}

var builtinTypes = map[string]struct{}{
	"time.Time":     {},
	"time.Duration": {},
}

func toTSType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "[]uint8":
		return "number[]"
	case "time.Time":
		return "string"
	default:
		if strings.HasPrefix(goType, "[]") {
			elemType := goType[2:]
			return toTSType(elemType) + "[]"
		}
		if strings.HasPrefix(goType, "map[") {
			// Extract key and value types from map[K]V
			parts := strings.Split(goType[4:], "]")
			if len(parts) == 2 {
				keyType := parts[0]
				valueType := parts[1]
				tsKeyType := "string"
				if keyType == "int" || keyType == "int64" || keyType == "uint" || keyType == "uint64" {
					tsKeyType = "number"
				}
				return "Record<" + tsKeyType + ", " + toTSType(valueType) + ">"
			}
		}
		if strings.HasPrefix(goType, "*") {
			return toTSType(goType[1:]) + " | undefined"
		}
		return goType
	}
}

// OpenAPI structures for generating OpenAPI specs
type OpenAPIInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type OpenAPISchema struct {
	Type                 string                    `yaml:"type,omitempty"`
	Properties           map[string]*OpenAPISchema `yaml:"properties,omitempty"`
	Items                *OpenAPISchema            `yaml:"items,omitempty"`
	AdditionalProperties *OpenAPISchema            `yaml:"additionalProperties,omitempty"`
	Required             []string                  `yaml:"required,omitempty"`
	Ref                  string                    `yaml:"$ref,omitempty"`
	Format               string                    `yaml:"format,omitempty"`
	MinLength            *int                      `yaml:"minLength,omitempty"`
	MaxLength            *int                      `yaml:"maxLength,omitempty"`
	Description          string                    `yaml:"description,omitempty"`
	Minimum              *int                      `yaml:"minimum,omitempty"`
	Maximum              *int                      `yaml:"maximum,omitempty"`
	Enum                 []string                  `yaml:"enum,omitempty"`
	Example              interface{}               `yaml:"example,omitempty"`
}

type OpenAPIParameter struct {
	Name        string         `yaml:"name"`
	In          string         `yaml:"in"`
	Description string         `yaml:"description,omitempty"`
	Required    bool           `yaml:"required"`
	Schema      *OpenAPISchema `yaml:"schema"`
	Example     interface{}    `yaml:"example,omitempty"`
}

type OpenAPIHeader struct {
	Description string         `yaml:"description,omitempty"`
	Required    bool           `yaml:"required,omitempty"`
	Schema      *OpenAPISchema `yaml:"schema"`
}

type OpenAPIMediaType struct {
	Schema *OpenAPISchema `yaml:"schema"`
}

type OpenAPIContent struct {
	ApplicationJSON *OpenAPIMediaType `yaml:"application/json,omitempty"`
}

type OpenAPIRequestBody struct {
	Content *OpenAPIContent `yaml:"content"`
}

type OpenAPIResponse struct {
	Description string                    `yaml:"description"`
	Content     *OpenAPIContent           `yaml:"content,omitempty"`
	Headers     map[string]*OpenAPIHeader `yaml:"headers,omitempty"`
}

type OpenAPIOperation struct {
	OperationID string                      `yaml:"operationId"`
	Description string                      `yaml:"description,omitempty"`
	Parameters  []*OpenAPIParameter         `yaml:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody         `yaml:"requestBody,omitempty"`
	Responses   map[string]*OpenAPIResponse `yaml:"responses"`
}

type OpenAPIPathItem struct {
	Get  *OpenAPIOperation `yaml:"get,omitempty"`
	Post *OpenAPIOperation `yaml:"post,omitempty"`
}

type OpenAPIComponents struct {
	Schemas map[string]*OpenAPISchema `yaml:"schemas"`
}

type OpenAPISpec struct {
	OpenAPI    string                      `yaml:"openapi"`
	Info       *OpenAPIInfo                `yaml:"info"`
	Paths      map[string]*OpenAPIPathItem `yaml:"paths"`
	Components *OpenAPIComponents          `yaml:"components"`
}

// GenerateOpenAPI generates an OpenAPI specification from the client metadata
func (g *ClientGen) GenerateOpenAPI(title, version string) (*OpenAPISpec, error) {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: &OpenAPIInfo{
			Title:   title,
			Version: version,
		},
		Paths: make(map[string]*OpenAPIPathItem),
		Components: &OpenAPIComponents{
			Schemas: make(map[string]*OpenAPISchema),
		},
	}

	// Collect all schemas from data types
	allSchemas := make(map[string]*OpenAPISchema)
	for _, api := range g.meta.Apis {
		for _, dataType := range api.DataTypes {
			schema := g.dataTypeToSchema(dataType)
			allSchemas[dataType.Name] = schema
		}
	}

	// Add paths and operations
	for _, api := range g.meta.Apis {
		path := "/" + api.OperationID
		pathItem := &OpenAPIPathItem{}

		operation := &OpenAPIOperation{
			OperationID: api.OperationID,
			Description: api.Spec.Description,
			Responses: map[string]*OpenAPIResponse{
				"200": {
					Description: "Success",
				},
			},
		}

		// Add request headers from spec
		if reqHeaders := g.specToRequestHeaders(api.Spec); reqHeaders != nil {
			operation.Parameters = append(operation.Parameters, reqHeaders...)
		}

		// Add response headers from spec
		if respHeaders := g.specToResponseHeaders(api.Spec); respHeaders != nil {
			operation.Responses["200"].Headers = respHeaders
		}

		// Add error responses from spec
		if errorResponses := g.specToErrorResponses(api.Spec); errorResponses != nil {
			for code, response := range errorResponses {
				operation.Responses[code] = response
			}
		}

		if api.Method == "GET" {
			// Handle GET parameters
			for _, field := range api.Input.Fields {
				if field.SchemaTag != "" {
					param := &OpenAPIParameter{
						Name:     field.SchemaTag,
						In:       "query",
						Required: true,
						Schema:   g.fieldToSchema(field),
					}
					operation.Parameters = append(operation.Parameters, param)
				}
			}

			// Add response body if output has fields
			if len(api.Output.Fields) > 0 {
				operation.Responses["200"].Content = &OpenAPIContent{
					ApplicationJSON: &OpenAPIMediaType{
						Schema: &OpenAPISchema{
							Ref: "#/components/schemas/" + api.Output.Name,
						},
					},
				}
			}

			pathItem.Get = operation
		} else {
			// Handle POST request body
			if len(api.Input.Fields) > 0 {
				operation.RequestBody = &OpenAPIRequestBody{
					Content: &OpenAPIContent{
						ApplicationJSON: &OpenAPIMediaType{
							Schema: &OpenAPISchema{
								Ref: "#/components/schemas/" + api.Input.Name,
							},
						},
					},
				}
			}

			// Add response body if output has fields
			if len(api.Output.Fields) > 0 {
				operation.Responses["200"].Content = &OpenAPIContent{
					ApplicationJSON: &OpenAPIMediaType{
						Schema: &OpenAPISchema{
							Ref: "#/components/schemas/" + api.Output.Name,
						},
					},
				}
			}

			pathItem.Post = operation
		}

		spec.Paths[path] = pathItem
	}

	// Add all schemas to components
	spec.Components.Schemas = allSchemas

	return spec, nil
}

func (g *ClientGen) dataTypeToSchema(dataType DataType) *OpenAPISchema {
	if len(dataType.Fields) == 0 {
		return nil
	}

	schema := &OpenAPISchema{
		Type:       "object",
		Properties: make(map[string]*OpenAPISchema),
		Required:   []string{},
	}

	for _, field := range dataType.Fields {
		propName := field.Name
		if field.JsonTag != "" {
			propName = field.JsonTag
		}

		schema.Properties[propName] = g.fieldToSchema(field)

		// Add to required if not a pointer type
		if !strings.HasPrefix(field.TypeName, "*") {
			schema.Required = append(schema.Required, propName)
		}
	}

	return schema
}

func (g *ClientGen) fieldToSchema(field Field) *OpenAPISchema {
	return g.typeNameToSchema(field.TypeName)
}

func (g *ClientGen) specToRequestHeaders(spec vel.Spec) []*OpenAPIParameter {
	if spec.RequestHeaders.Key == "" {
		return nil
	}

	param := &OpenAPIParameter{
		Name:        spec.RequestHeaders.Key,
		In:          "header",
		Description: spec.RequestHeaders.Description,
		Required:    spec.RequestHeaders.Validation.Required,
		Schema:      g.primitiveTypeToSchemaWithValidation(spec.RequestHeaders.ValueType, spec.RequestHeaders.Validation),
	}

	if spec.RequestHeaders.ValueExample != "" {
		param.Example = spec.RequestHeaders.ValueExample
	}

	return []*OpenAPIParameter{param}
}

func (g *ClientGen) specToResponseHeaders(spec vel.Spec) map[string]*OpenAPIHeader {
	if spec.ResponseHeaders.Key == "" {
		return nil
	}

	headers := make(map[string]*OpenAPIHeader)
	headers[spec.ResponseHeaders.Key] = &OpenAPIHeader{
		Description: spec.ResponseHeaders.Description,
		Required:    spec.ResponseHeaders.Validation.Required,
		Schema:      g.primitiveTypeToSchemaWithValidation(spec.ResponseHeaders.ValueType, spec.ResponseHeaders.Validation),
	}

	return headers
}

func (g *ClientGen) primitiveTypeToSchema(primitiveType vel.PrimitiveType) *OpenAPISchema {
	return g.primitiveTypeToSchemaWithValidation(primitiveType, vel.Validation{})
}

func (g *ClientGen) primitiveTypeToSchemaWithValidation(primitiveType vel.PrimitiveType, validation vel.Validation) *OpenAPISchema {
	schema := &OpenAPISchema{}

	switch primitiveType {
	case vel.String:
		schema.Type = "string"
	case vel.Int:
		schema.Type = "integer"
	case vel.Uint:
		schema.Type = "integer"
	case vel.Float64:
		schema.Type = "number"
	case vel.Bool:
		schema.Type = "boolean"
	default:
		schema.Type = "string"
	}

	// Add validation constraints
	if validation.MinLen > 0 {
		schema.MinLength = &validation.MinLen
	}
	if validation.MaxLen > 0 {
		schema.MaxLength = &validation.MaxLen
	}
	if validation.MinValue > 0 {
		schema.Minimum = &validation.MinValue
	}
	if validation.MaxValue > 0 {
		max := int(validation.MaxValue)
		schema.Maximum = &max
	}
	if len(validation.Enum) > 0 {
		schema.Enum = validation.Enum
	}

	return schema
}

func (g *ClientGen) specToErrorResponses(spec vel.Spec) map[string]*OpenAPIResponse {
	if len(spec.Errors) == 0 {
		return nil
	}

	responses := make(map[string]*OpenAPIResponse)
	for httpStatus, errorSpecs := range spec.Errors {
		errorCodes := make([]string, 0, len(errorSpecs))
		descriptions := make([]string, 0, len(errorSpecs))
		
		// Collect all meta properties from all error specs for this status
		allMetaProperties := make(map[string]*OpenAPISchema)
		
		for _, errorSpec := range errorSpecs {
			errorCodes = append(errorCodes, errorSpec.Code)
			descriptions = append(descriptions, fmt.Sprintf("* `%s` - %s", errorSpec.Code, errorSpec.Description))
			
			// Merge meta properties
			metaProps := g.errorMetaToProperties(errorSpec.Meta)
			for key, value := range metaProps {
				allMetaProperties[key] = value
			}
		}
		
		httpStatusStr := fmt.Sprintf("%d", httpStatus)
		
		// Create consolidated description
		consolidatedDescription := "Error codes:\n  " + strings.Join(descriptions, "\n  ")
		
		responses[httpStatusStr] = &OpenAPIResponse{
			Description: consolidatedDescription,
			Content: &OpenAPIContent{
				ApplicationJSON: &OpenAPIMediaType{
					Schema: &OpenAPISchema{
						Type: "object",
						Properties: map[string]*OpenAPISchema{
							"code": {
								Type: "string",
								Enum: errorCodes,
							},
							"message": {
								Type: "string",
							},
							"meta": {
								Type:       "object",
								Properties: allMetaProperties,
							},
						},
						Required: []string{"code"},
					},
				},
			},
		}
	}

	return responses
}

func (g *ClientGen) errorMetaToProperties(meta []vel.KeyValueSpec) map[string]*OpenAPISchema {
	if len(meta) == 0 {
		return nil
	}

	properties := make(map[string]*OpenAPISchema)
	for _, m := range meta {
		schema := &OpenAPISchema{}

		// Set type first
		switch m.ValueType {
		case vel.String:
			schema.Type = "string"
		case vel.Int:
			schema.Type = "integer"
		case vel.Uint:
			schema.Type = "integer"
		case vel.Float64:
			schema.Type = "number"
		case vel.Bool:
			schema.Type = "boolean"
		default:
			schema.Type = "string"
		}

		// Add validation constraints first
		if m.Validation.MinLen > 0 {
			schema.MinLength = &m.Validation.MinLen
		}
		if m.Validation.MaxLen > 0 {
			schema.MaxLength = &m.Validation.MaxLen
		}
		if m.Validation.MinValue > 0 {
			schema.Minimum = &m.Validation.MinValue
		}
		if m.Validation.MaxValue > 0 {
			max := int(m.Validation.MaxValue)
			schema.Maximum = &max
		}
		if len(m.Validation.Enum) > 0 {
			schema.Enum = m.Validation.Enum
		}
		
		// Add description after all validation constraints
		if m.Description != "" {
			schema.Description = m.Description
		}

		properties[m.Key] = schema
	}

	return properties
}

func (g *ClientGen) typeNameToSchema(typeName string) *OpenAPISchema {
	switch typeName {
	case "string":
		return &OpenAPISchema{Type: "string"}
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return &OpenAPISchema{Type: "integer"}
	case "float32", "float64":
		return &OpenAPISchema{Type: "number"}
	case "bool":
		return &OpenAPISchema{Type: "boolean"}
	case "[]uint8":
		return &OpenAPISchema{
			Type:  "array",
			Items: &OpenAPISchema{Type: "integer"},
		}
	case "time.Time":
		return &OpenAPISchema{
			Type:   "string",
			Format: "date-time",
		}
	}

	// Handle arrays
	if strings.HasPrefix(typeName, "[]") {
		elemType := typeName[2:]
		return &OpenAPISchema{
			Type:  "array",
			Items: g.typeNameToSchema(elemType),
		}
	}

	// Handle maps
	if strings.HasPrefix(typeName, "map[") {
		parts := strings.Split(typeName[4:], "]")
		if len(parts) == 2 {
			valueType := parts[1]
			return &OpenAPISchema{
				Type:                 "object",
				AdditionalProperties: g.typeNameToSchema(valueType),
			}
		}
	}

	// Handle pointers - remove the * and reference the type
	if strings.HasPrefix(typeName, "*") {
		return &OpenAPISchema{
			Ref: "#/components/schemas/" + typeName[1:],
		}
	}

	// Reference to another schema
	return &OpenAPISchema{
		Ref: "#/components/schemas/" + typeName,
	}
}

// GenerateOpenAPIYAML generates OpenAPI YAML from the client metadata
func (g *ClientGen) GenerateOpenAPIYAML(w io.Writer, title, version string) error {
	spec, err := g.GenerateOpenAPI(title, version)
	if err != nil {
		return err
	}

	// Create a custom node to force double quotes
	node := &yaml.Node{}
	err = node.Encode(spec)
	if err != nil {
		return err
	}

	// Force double quotes for all string values
	forceDoubleQuotes(node)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	err = encoder.Encode(node)
	if err != nil {
		return err
	}

	yamlBytes := buf.Bytes()

	_, err = w.Write(yamlBytes)
	return err
}

// forceDoubleQuotes recursively forces double quotes on specific string values
func forceDoubleQuotes(node *yaml.Node) {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		// Only quote specific values that need quotes
		if node.Value == "200" || strings.HasPrefix(node.Value, "#/components/schemas/") {
			node.Style = yaml.DoubleQuotedStyle
		}
	}

	for _, child := range node.Content {
		forceDoubleQuotes(child)
	}
}
