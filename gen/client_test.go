package gen

import (
	"bytes"
	_ "embed"
	"testing"
	"time"

	"github.com/dennypenta/vel"
)

//go:embed testdata/test.go
var infoClientOutputGo string

//go:embed testdata/test.ts
var infoClientOutputTs string

//go:embed testdata/openapi.yaml
var expectedOpenAPIYAML string

type TestTypeNoJsonTags struct {
	Value string
}

type TestTypeNestedTypes struct {
	Data  TestStruct `json:"data"`
	Chunk []byte     `json:"chunk"`
	// High level elements
	NextLevelSlice   []HighElem          `json:"slice"`
	Map              map[int]HighMapElem `json:"map"`
	NextLevelNestedP *HighPointer        `json:"nextP"`
}

type TestStruct struct {
	Row              int                   `json:"row"`
	Line             string                `json:"line"`
	NextLevelNested  TestNextLevelStruct   `json:"next"`
	NextLevelSlice   []TestNextLevelElem   `json:"slice"`
	Map              map[int]MapValue      `json:"map"`
	NextLevelNestedP *TestNextLevelStructP `json:"nextP"`
	// TODO: Highlight as not supported (who ever might need them??? )
	// NextLevelSliceP  []*TestNextLevelElemP `json:"sliceP"`
	// MapP             map[int]*MapValueP    `json:"mapP"`
}

type TestNextLevelStruct struct {
	Extra string `json:"extra"`
}

type TestNextLevelElem struct {
	Int int `json:"int"`
}
type MapValue struct {
	Value string
}
type TestNextLevelStructP struct {
	Extra string `json:"extra"`
}
type HighElem struct {
	Int int `json:"int"`
}
type HighMapElem struct {
	Value string
}
type HighPointer struct {
	Extra string `json:"extra"`
}

type Empty struct{}

// Custom assertion functions to replace testify
func assertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

type GetQuery struct {
	Value string `schema:"value"`
	Field int    `schema:"field"`
}

type GetResp struct {
	Getting int
}

type TimeTestRequest struct {
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`
}

type TimeTestResponse struct {
	ProcessedAt time.Time `json:"processedAt"`
	ID          string    `json:"id"`
}

func TestGenClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skip: requires goimports installation")
	}

	type testCase struct {
		name           string
		expected       string
		templateName   string
		postProcessing string
	}

	for _, tc := range []testCase{
		{
			name:           "go",
			expected:       infoClientOutputGo,
			templateName:   "go:default",
			postProcessing: "goimports",
		},
		{
			name:           "ts",
			expected:       infoClientOutputTs,
			templateName:   "ts:default",
			postProcessing: "prettier --parser typescript",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}

			gener, err := New(ClientDesc{
				TypeName:    "Client",
				PackageName: "client",
			}, []vel.HandlerMeta{
				{Input: TestTypeNoJsonTags{}, Output: TestTypeNoJsonTags{}, OperationID: "test1", Method: "POST"},
				{Input: TestTypeNestedTypes{}, Output: TestTypeNestedTypes{}, OperationID: "test2", Method: "POST"},
				{Input: struct{}{}, Output: Empty{}, OperationID: "testEmpty", Method: "POST"},
				{Input: GetQuery{}, Output: GetResp{}, OperationID: "testGet", Method: "GET"},
				{Input: TimeTestRequest{}, Output: TimeTestResponse{}, OperationID: "testTime", Method: "POST"},
			})
			requireNoError(t, err)
			err = gener.Generate(buf, tc.templateName, tc.postProcessing)
			requireNoError(t, err)
			assertEqual(t, tc.expected, buf.String())

			// optional: step to visualize the diff
			// f, err := os.OpenFile("./testdata/out.go", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
			// requireNoError(t, err)
			// defer f.Close()

			// _, err = f.Write(buf.Bytes())
			// requireNoError(t, err)
		})
	}
}

func TestGenOpenAPI(t *testing.T) {
	buf := &bytes.Buffer{}

	gener, err := New(ClientDesc{
		TypeName:    "Client",
		PackageName: "client",
	}, []vel.HandlerMeta{
		{
			Input: TestTypeNoJsonTags{}, Output: TestTypeNoJsonTags{}, OperationID: "test1", Method: "POST", Spec: vel.Spec{
				Description: "Test endpoint with headers",
				RequestHeaders: vel.KeyValueSpec{
					Key:          "X-API-Key",
					ValueType:    vel.String,
					Description:  "API key for authentication",
					ValueExample: "treenq_12341234",
					Validation: vel.Validation{
						Required: true,
					},
				},
				ResponseHeaders: vel.KeyValueSpec{
					Key:         "X-Rate-Limit",
					ValueType:   vel.Int,
					Description: "Rate limit remaining",
					Validation: vel.Validation{
						Required: false,
						MinValue: 1,
						MaxValue: 3,
						Enum:     []string{"1", "2", "3"},
					},
				},
				Errors: map[int][]vel.ErrorSpec{
					400: {{
						Code:        "ERROR_CODE",
						Description: "meaningful text",
						Meta: []vel.KeyValueSpec{
							{
								Key:         "field",
								ValueType:   vel.String,
								Description: "some field",
								Validation: vel.Validation{
									Required: false,
									MinLen:   1,
									MaxLen:   300,
								},
							},
						},
					}, {
						Code:        "ANOTHER_CODE",
						Description: "some text",
						Meta:        []vel.KeyValueSpec{},
					}},
					450: {{
						Code:        "ERROR_CODE_450",
						Description: "meaningful text",
						Meta: []vel.KeyValueSpec{
							{
								Key:         "field",
								ValueType:   vel.String,
								Description: "some field",
								Validation: vel.Validation{
									Required: false,
									MinLen:   1,
									MaxLen:   300,
								},
							},
						},
					}},
				},
			},
		},
		{Input: TestTypeNestedTypes{}, Output: TestTypeNestedTypes{}, OperationID: "test2", Method: "POST"},
		{Input: struct{}{}, Output: Empty{}, OperationID: "testEmpty", Method: "POST"},
		{Input: GetQuery{}, Output: GetResp{}, OperationID: "testGet", Method: "GET"},
		{Input: TimeTestRequest{}, Output: TimeTestResponse{}, OperationID: "testTime", Method: "POST"},
	})
	requireNoError(t, err)

	err = gener.GenerateOpenAPIYAML(buf, "Test API", "1.0.0")
	requireNoError(t, err)

	assertEqual(t, expectedOpenAPIYAML, buf.String())
}
