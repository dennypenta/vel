package simple

import (
	"context"
	"net/http"

	"github.com/dennypenta/vel"
)

type HelloRequest struct {
	Name string `json:"name"`
}

type HelloResponse struct {
	Message string `json:"message"`
}

func HelloHandler(ctx context.Context, req HelloRequest) (HelloResponse, *vel.Error) {
	return HelloResponse{
		Message: "Hello, " + req.Name,
	}, nil
}

func NewRouter() *vel.Router {
	router := vel.NewRouter()
	vel.RegisterPost(router, "hello", HelloHandler)
	return router
}

func main() {
	router := NewRouter()
	http.ListenAndServe(":8080", router.Mux())
}
