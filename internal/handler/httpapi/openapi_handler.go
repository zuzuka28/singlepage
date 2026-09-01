package httpapi

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"singlepage/internal/handler/httpapi/gen"
)

func newOpenAPIHandler(pages pageService) http.Handler {
	implementation := &handler{pages: pages}
	strict := gen.NewStrictHandlerWithOptions(
		implementation,
		nil,
		gen.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  writeRequestError,
			ResponseErrorHandlerFunc: writeResponseError,
		},
	)
	validator := nethttpmiddleware.OapiRequestValidatorWithOptions(
		mustSpecification(),
		&nethttpmiddleware.Options{
			Options: openapi3filter.Options{
				ExcludeRequestBody:          false,
				ExcludeRequestQueryParams:   false,
				ExcludeResponseBody:         false,
				ExcludeReadOnlyValidations:  false,
				ExcludeWriteOnlyValidations: false,
				IncludeResponseStatus:       false,
				MultiError:                  false,
				RegexCompiler:               nil,
				AuthenticationFunc:          openapi3filter.NoopAuthenticationFunc,
				SkipSettingDefaults:         false,
			},
			ErrorHandler:          nil,
			ErrorHandlerWithOpts:  writeValidationError,
			MultiErrorHandler:     nil,
			SilenceServersWarning: true,
			DoNotValidateServers:  true,
		},
	)

	return gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseURL:          "",
		BaseRouter:       nil,
		Middlewares:      []gen.MiddlewareFunc{validator, requestRoute},
		ErrorHandlerFunc: writeRequestError,
	})
}
