package config

import "time"

const (
	defaultListen            = "127.0.0.1:8080"
	defaultMetricsListen     = "127.0.0.1:9090"
	defaultSQLiteDSN         = "data.db"
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 20 * time.Second
	defaultWriteTimeout      = 20 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
	defaultMaxDatabaseBytes  = 512 << 20
	defaultMaxPages          = 100_000
	defaultMaxRequestBody    = 16 << 20
	defaultCreateRate        = 1
	defaultCreateBurst       = 20
	defaultAdminRate         = 1
	defaultAdminBurst        = 5
	maxAdminTokenBytes       = 256
	minAdminTokenBytes       = 32
)

func defaults() Config {
	return Config{
		HTTP: HTTP{
			Listen:            defaultListen,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
			ReadTimeout:       defaultReadTimeout,
			WriteTimeout:      defaultWriteTimeout,
			IdleTimeout:       defaultIdleTimeout,
			ShutdownTimeout:   defaultShutdownTimeout,
		},
		Metrics: Metrics{Listen: defaultMetricsListen},
		SQLite:  SQLite{DSN: defaultSQLiteDSN, MaxBytes: defaultMaxDatabaseBytes},
		Page:    Page{MaxPages: defaultMaxPages},
		Protection: Protection{
			MaxRequestBodyBytes: defaultMaxRequestBody,
			CreateRatePerSecond: defaultCreateRate,
			CreateBurst:         defaultCreateBurst,
			AdminRatePerSecond:  defaultAdminRate,
			AdminBurst:          defaultAdminBurst,
			TrustProxyHeaders:   false,
			AdminToken:          "",
		},
		CORS: CORS{
			AllowedOrigins: []string{},
			AllowedMethods: []string{
				"GET",
				"HEAD",
				"POST",
				"PUT",
				"DELETE",
				"OPTIONS",
			},
			AllowedHeaders:   []string{"Authorization", "Content-Type"},
			ExposedHeaders:   []string{"X-Request-ID"},
			AllowCredentials: false,
			MaxAge:           0,
		},
	}
}

func clone(values []string) []string {
	return append([]string(nil), values...)
}
