package config

import (
	"fmt"
	"os"
	"strings"
)

// Load reads and validates configuration from SINGLEPAGE_* environment variables.
func Load() (Config, error) {
	loaded := defaults()

	loadString(&loaded.HTTP.Listen, envHTTPListen)
	loadString(&loaded.Metrics.Listen, envMetricsListen)
	loadString(&loaded.SQLite.DSN, envSQLiteDSN)

	err := loadDurations(&loaded)
	if err != nil {
		return Config{}, err
	}

	err = loadNumbers(&loaded)
	if err != nil {
		return Config{}, err
	}

	err = loadBooleans(&loaded)
	if err != nil {
		return Config{}, err
	}

	loadLists(&loaded)

	err = loadAdminToken(&loaded)
	if err != nil {
		return Config{}, err
	}

	err = validate(loaded)
	if err != nil {
		return Config{}, err
	}

	return loaded, nil
}

func loadAdminToken(loaded *Config) error {
	path := strings.TrimSpace(os.Getenv(envAdminTokenFile))
	if path == "" {
		return nil
	}

	raw, err := os.ReadFile(path) // #nosec G703 -- The operator explicitly configures this secret file.
	if err != nil {
		return fmt.Errorf("%w: read %s: %w", errInvalidEnvironment, envAdminTokenFile, err)
	}

	token := strings.TrimSpace(string(raw))
	if token == "" {
		return invalid(envAdminTokenFile, "token file must not be empty")
	}

	loaded.Protection.AdminToken = token

	return nil
}

func loadString(target *string, name string) {
	if raw := strings.TrimSpace(os.Getenv(name)); raw != "" {
		*target = raw
	}
}
