// Package config loads and validates the application's environment configuration.
package config

// Config contains all runtime configuration, grouped by application subsystem.
type Config struct {
	HTTP       HTTP
	Metrics    Metrics
	SQLite     SQLite
	Page       Page
	Protection Protection
	CORS       CORS
}

// Load reads every configuration block from SINGLEPAGE_* environment variables.
func Load() (Config, error) {
	loaded := defaultConfig()

	err := loadHTTP(&loaded.HTTP)
	if err != nil {
		return Config{}, err
	}

	loadMetrics(&loaded.Metrics)

	err = loadSQLite(&loaded.SQLite)
	if err != nil {
		return Config{}, err
	}

	err = loadPage(&loaded.Page)
	if err != nil {
		return Config{}, err
	}

	err = loadProtection(&loaded.Protection)
	if err != nil {
		return Config{}, err
	}

	err = loadCORS(&loaded.CORS)
	if err != nil {
		return Config{}, err
	}

	err = validateConfig(loaded)
	if err != nil {
		return Config{}, err
	}

	return loaded, nil
}

// LoadStorage reads only storage-related blocks. Native applications use it so
// irrelevant HTTP environment variables cannot prevent startup.
func LoadStorage() (Storage, error) {
	storage := defaultStorage()

	err := loadSQLite(&storage.SQLite)
	if err != nil {
		return Storage{}, err
	}

	err = loadPage(&storage.Page)
	if err != nil {
		return Storage{}, err
	}

	err = validateStorage(storage)
	if err != nil {
		return Storage{}, err
	}

	return storage, nil
}

func defaultConfig() Config {
	return Config{
		HTTP:       defaultHTTP(),
		Metrics:    defaultMetrics(),
		SQLite:     defaultSQLite(),
		Page:       defaultPage(),
		Protection: defaultProtection(),
		CORS:       defaultCORS(),
	}
}

func validateConfig(loaded Config) error {
	err := validateHTTP(loaded.HTTP)
	if err != nil {
		return err
	}

	err = validateMetrics(loaded.HTTP, loaded.Metrics)
	if err != nil {
		return err
	}

	err = validateStorage(loaded.Storage())
	if err != nil {
		return err
	}

	err = validateProtection(loaded.Protection)
	if err != nil {
		return err
	}

	return validateCORS(loaded.CORS)
}
