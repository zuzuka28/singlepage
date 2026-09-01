package httpapi

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"

	"singlepage/internal/handler/httpapi/gen"
)

func mustSpecification() *openapi3.T {
	specification, err := gen.GetSwagger()
	if err != nil {
		panic(fmt.Errorf("load embedded OpenAPI specification: %w", err))
	}

	return specification
}
