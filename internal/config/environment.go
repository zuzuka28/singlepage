package config

const (
	envHTTPListen            = "SINGLEPAGE_HTTP_LISTEN"
	envHTTPReadHeaderTimeout = "SINGLEPAGE_HTTP_READ_HEADER_TIMEOUT"
	envHTTPReadTimeout       = "SINGLEPAGE_HTTP_READ_TIMEOUT"
	envHTTPWriteTimeout      = "SINGLEPAGE_HTTP_WRITE_TIMEOUT"
	envHTTPIdleTimeout       = "SINGLEPAGE_HTTP_IDLE_TIMEOUT"
	envHTTPShutdownTimeout   = "SINGLEPAGE_HTTP_SHUTDOWN_TIMEOUT"
	envMetricsListen         = "SINGLEPAGE_METRICS_LISTEN"
	envSQLiteDSN             = "SINGLEPAGE_SQLITE_DSN"
	envSQLiteMaxBytes        = "SINGLEPAGE_SQLITE_MAX_BYTES"
	envPageMaxPages          = "SINGLEPAGE_PAGE_MAX_PAGES"
	envRequestMaxBody        = "SINGLEPAGE_REQUEST_MAX_BODY_BYTES"
	envCreateRate            = "SINGLEPAGE_CREATE_RATE_PER_SECOND"
	envCreateBurst           = "SINGLEPAGE_CREATE_BURST"
	envAdminRate             = "SINGLEPAGE_ADMIN_RATE_PER_SECOND"
	envAdminBurst            = "SINGLEPAGE_ADMIN_BURST"
	envTrustProxy            = "SINGLEPAGE_TRUST_PROXY_HEADERS"
	envAdminTokenFile        = "SINGLEPAGE_ADMIN_TOKEN_FILE"
	envCORSAllowedOrigins    = "SINGLEPAGE_CORS_ALLOWED_ORIGINS"
	envCORSAllowedMethods    = "SINGLEPAGE_CORS_ALLOWED_METHODS"
	envCORSAllowedHeaders    = "SINGLEPAGE_CORS_ALLOWED_HEADERS"
	envCORSExposedHeaders    = "SINGLEPAGE_CORS_EXPOSED_HEADERS"
	envCORSAllowCredentials  = "SINGLEPAGE_CORS_ALLOW_CREDENTIALS" //nolint:gosec // Environment name.
	envCORSMaxAge            = "SINGLEPAGE_CORS_MAX_AGE"
)
