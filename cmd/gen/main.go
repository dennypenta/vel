package main

import (
	"log"
	"os"

	"github.com/dennypenta/vel/examples/simple"
	"github.com/dennypenta/vel/gen"
)

func main() {
	router := simple.NewRouter()

	err := gen.GenerateClient(router, os.Stdout, gen.ClientGeneratorConfig{
		TypeName:    "Client",
		PackageName: "client",
		Language:    "go",
		PostProcess: "goimports",
	})
	if err != nil {
		log.Fatalf("Failed to generate Go client: %s\n", err.Error())
	}

	err = gen.GenerateOpenAPI(router, os.Stdout, "Simple API", "1.0.0")
	if err != nil {
		log.Fatalf("Failed to generate OpenAPI: %s\n", err.Error())
	}
}
