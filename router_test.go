package vel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TestRequest struct {
	Message string `json:"message"`
}

type TestResponse struct {
	Reply string `json:"reply"`
}

type request struct {
	method   string
	path     string
	body     string
	wantCode int
	wantBody string
}

func TestRouterSubrouters(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Router
		requests []request
	}{
		{
			name: "basic router (no subrouter)",
			setup: func() *Router {
				r := NewRouter()
				RegisterGet(r, "users", func(ctx context.Context, req struct{}) (TestResponse, *Error) {
					return TestResponse{Reply: "users list"}, nil
				})
				RegisterPost(r, "users", func(ctx context.Context, req TestRequest) (TestResponse, *Error) {
					return TestResponse{Reply: "user created: " + req.Message}, nil
				})
				return r
			},
			requests: []request{
				{"GET", "/users", "", 200, `{"reply":"users list"}`},
				{"POST", "/users", `{"message":"john"}`, 200, `{"reply":"user created: john"}`},
				{"GET", "/v1/users", "", 404, ""},
			},
		},
		{
			name: "basic router + /v1 subrouter",
			setup: func() *Router {
				r := NewRouter()
				RegisterGet(r, "health", func(ctx context.Context, req struct{}) (TestResponse, *Error) {
					return TestResponse{Reply: "ok"}, nil
				})

				v1 := r.Subrouter("v1")
				RegisterGet(v1, "users", func(ctx context.Context, req struct{}) (TestResponse, *Error) {
					return TestResponse{Reply: "v1 users list"}, nil
				})
				RegisterPost(v1, "users", func(ctx context.Context, req TestRequest) (TestResponse, *Error) {
					return TestResponse{Reply: "v1 user created: " + req.Message}, nil
				})
				return r
			},
			requests: []request{
				{"GET", "/health", "", 200, `{"reply":"ok"}`},
				{"GET", "/v1/users", "", 200, `{"reply":"v1 users list"}`},
				{"POST", "/v1/users", `{"message":"jane"}`, 200, `{"reply":"v1 user created: jane"}`},
				{"GET", "/users", "", 404, ""},
			},
		},
		{
			name: "v1 subrouter + v2 subrouter (no basic routes)",
			setup: func() *Router {
				r := NewRouter()

				v1 := r.Subrouter("v1")
				RegisterGet(v1, "users", func(ctx context.Context, req struct{}) (TestResponse, *Error) {
					return TestResponse{Reply: "v1 users"}, nil
				})
				RegisterPost(v1, "posts", func(ctx context.Context, req TestRequest) (TestResponse, *Error) {
					return TestResponse{Reply: "v1 post: " + req.Message}, nil
				})

				v2 := r.Subrouter("v2")
				RegisterGet(v2, "users", func(ctx context.Context, req struct{}) (TestResponse, *Error) {
					return TestResponse{Reply: "v2 users"}, nil
				})
				RegisterPost(v2, "posts", func(ctx context.Context, req TestRequest) (TestResponse, *Error) {
					return TestResponse{Reply: "v2 post: " + req.Message}, nil
				})
				return r
			},
			requests: []request{
				{"GET", "/v1/users", "", 200, `{"reply":"v1 users"}`},
				{"POST", "/v1/posts", `{"message":"hello v1"}`, 200, `{"reply":"v1 post: hello v1"}`},
				{"GET", "/v2/users", "", 200, `{"reply":"v2 users"}`},
				{"POST", "/v2/posts", `{"message":"hello v2"}`, 200, `{"reply":"v2 post: hello v2"}`},
				{"GET", "/users", "", 404, ""},
				{"GET", "/posts", "", 404, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := tt.setup()
			server := httptest.NewServer(router.Mux())
			defer server.Close()

			for _, req := range tt.requests {
				t.Run(fmt.Sprintf("%s %s", req.method, req.path), func(t *testing.T) {
					var resp *http.Response
					var err error

					r, err := http.NewRequest(req.method, server.URL+req.path, strings.NewReader(req.body))
					if err != nil {
						t.Fatalf("failed to build request: %v", err)
					}
					resp, err = http.DefaultClient.Do(r)
					if err != nil {
						t.Fatalf("request failed: %v", err)
					}
					defer resp.Body.Close()

					if resp.StatusCode != req.wantCode {
						t.Errorf("expected status %d, got %d", req.wantCode, resp.StatusCode)
					}

					if req.wantBody != "" {
						var got TestResponse
						if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
							t.Fatalf("failed to decode response: %v", err)
						}

						expectedResp := TestResponse{}
						if err := json.Unmarshal([]byte(req.wantBody), &expectedResp); err != nil {
							t.Fatalf("failed to unmarshal expected response: %v", err)
						}

						if got.Reply != expectedResp.Reply {
							t.Errorf("expected body %s, got %s", req.wantBody, fmt.Sprintf(`{"reply":"%s"}`, got.Reply))
						}
					}
				})
			}
		})
	}
}

func TestSubrouterMetadata(t *testing.T) {
	r := NewRouter()

	RegisterGet(r, "root", func(ctx context.Context, req struct{}) (struct{}, *Error) {
		return struct{}{}, nil
	})

	v1 := r.Subrouter("v1")
	RegisterGet(v1, "users", func(ctx context.Context, req struct{}) (struct{}, *Error) {
		return struct{}{}, nil
	})

	v2 := r.Subrouter("v2")
	RegisterPost(v2, "posts", func(ctx context.Context, req struct{}) (struct{}, *Error) {
		return struct{}{}, nil
	})

	rootMeta := r.Meta()
	if len(rootMeta) != 1 {
		t.Errorf("expected 1 handler in root, got %d", len(rootMeta))
	}
	if rootMeta[0].OperationID != "root" {
		t.Errorf("expected root operation, got %s", rootMeta[0].OperationID)
	}

	v1Meta := v1.Meta()
	if len(v1Meta) != 1 {
		t.Errorf("expected 1 handler in v1, got %d", len(v1Meta))
	}
	if v1Meta[0].OperationID != "users" {
		t.Errorf("expected users operation, got %s", v1Meta[0].OperationID)
	}

	v2Meta := v2.Meta()
	if len(v2Meta) != 1 {
		t.Errorf("expected 1 handler in v2, got %d", len(v2Meta))
	}
	if v2Meta[0].OperationID != "posts" {
		t.Errorf("expected posts operation, got %s", v2Meta[0].OperationID)
	}
}
