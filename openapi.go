package vel

type PrimitiveType string

const (
	Bool    PrimitiveType = "bool"
	Int     PrimitiveType = "int"
	Uint    PrimitiveType = "uint"
	Float64 PrimitiveType = "float64"
	// Array   Type = "array"
	// Map     Type = "map"
	String PrimitiveType = "string"
	// Object  Type = "object"
)

type Spec struct {
	Description     string
	RequestHeaders  KeyValueSpec
	ResponseHeaders KeyValueSpec
	Errors          map[int][]ErrorSpec
}

type ErrorSpec struct {
	Code        string
	Description string
	Meta        []KeyValueSpec
}

type KeyValueSpec struct {
	Key          string
	ValueType    PrimitiveType
	ValueExample string
	Description  string
	Validation   Validation
}

type Validation struct {
	Required bool
	MinLen   int
	MaxLen   int
	MinValue int
	MaxValue uint
	Enum     []string
}
