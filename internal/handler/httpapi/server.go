package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rs/cors"

	"singlepage/internal/config"
	"singlepage/internal/handler/frontend"
	"singlepage/internal/metrics"
	"singlepage/internal/middleware"
	modelpage "singlepage/internal/model/page"
)

// New creates the complete public application HTTP handler.
func New(
	pages pageService,
	applicationConfig config.Config,
	applicationMetrics *metrics.Metrics,
	logger *slog.Logger,
) http.Handler {
	apiHandler := buildAPIHandler(pages, applicationConfig, applicationMetrics)
	root := http.NewServeMux()
	root.Handle("/api/", apiHandler)
	root.Handle("/", frontend.New())

	return middleware.Chain(
		root,
		middleware.RequestID,
		middleware.RequestLogger(logger),
		middleware.Recovery,
		middleware.SecurityHeaders,
	)
}

func buildAPIHandler(
	pages pageService,
	applicationConfig config.Config,
	applicationMetrics *metrics.Metrics,
) http.Handler {
	apiHandler := middleware.Chain(
		newOpenAPIHandler(pages),
		apiMiddlewares(applicationConfig.Protection)...,
	)
	apiHandler = withCORS(apiHandler, applicationConfig.CORS)
	apiHandler = middleware.Recovery(apiHandler)

	return applicationMetrics.Middleware(apiHandler)
}

func apiMiddlewares(protection config.Protection) []middleware.Middleware {
	result := make([]middleware.Middleware, 0, 5)

	if protection.CreateRatePerSecond > 0 {
		createLimiter := middleware.NewClientRateLimiter(
			protection.CreateRatePerSecond,
			protection.CreateBurst,
		)
		result = append(result, middleware.When(
			isPageCreate,
			middleware.RateLimit(
				createLimiter,
				protection.TrustProxyHeaders,
			),
		))
	}

	adminAuth := middleware.BearerAuth(protection.AdminToken)
	if protection.AdminToken != "" && protection.AdminRatePerSecond > 0 {
		adminLimiter := middleware.NewClientRateLimiter(
			protection.AdminRatePerSecond,
			protection.AdminBurst,
		)
		result = append(result, middleware.When(
			isValidAdminDelete,
			middleware.RateLimit(
				adminLimiter,
				protection.TrustProxyHeaders,
			),
		))
	}

	return append(
		result,
		middleware.When(isValidAdminDelete, adminAuth),
		middleware.MaxBodyBytes(protection.MaxRequestBodyBytes),
		middleware.BearerToken,
	)
}

func withCORS(handler http.Handler, corsConfig config.CORS) http.Handler {
	if len(corsConfig.AllowedOrigins) == 0 {
		return handler
	}

	return cors.New(cors.Options{
		AllowedOrigins:             corsConfig.AllowedOrigins,
		AllowOriginFunc:            nil,
		AllowOriginRequestFunc:     nil,
		AllowOriginVaryRequestFunc: nil,
		AllowedMethods:             corsConfig.AllowedMethods,
		AllowedHeaders:             corsConfig.AllowedHeaders,
		ExposedHeaders:             corsConfig.ExposedHeaders,
		MaxAge:                     int(corsConfig.MaxAge / time.Second),
		AllowCredentials:           corsConfig.AllowCredentials,
		AllowPrivateNetwork:        false,
		OptionsPassthrough:         false,
		OptionsSuccessStatus:       http.StatusNoContent,
		Debug:                      false,
		Logger:                     nil,
	}).Handler(handler)
}

func isPageCreate(request *http.Request) bool {
	return request.Method == http.MethodPost && request.URL.Path == "/api/pages"
}

func isValidAdminDelete(request *http.Request) bool {
	if request.Method != http.MethodDelete {
		return false
	}

	id, found := strings.CutPrefix(request.URL.Path, "/api/admin/pages/")
	if !found || strings.ContainsRune(id, '/') {
		return false
	}

	return modelpage.ValidID(id)
}
