package config

import "strings"

const (
	envSQLiteDSN      = "SINGLEPAGE_SQLITE_DSN"
	envSQLiteMaxBytes = "SINGLEPAGE_SQLITE_MAX_BYTES"
	envPageMaxPages   = "SINGLEPAGE_PAGE_MAX_PAGES"

	defaultSQLiteDSN        = "data.db"
	defaultMaxDatabaseBytes = 512 << 20
	defaultMaxPages         = 100_000
)

// SQLite contains SQLite storage settings.
type SQLite struct {
	DSN      string
	MaxBytes int64
}

// Page contains limits applied to encrypted pages.
type Page struct {
	MaxPages int64
}

// Storage contains the transport-independent persistence configuration.
type Storage struct {
	SQLite SQLite
	Page   Page
}

// Storage returns the persistence subset of the complete configuration.
func (config Config) Storage() Storage {
	return Storage{SQLite: config.SQLite, Page: config.Page}
}

func defaultSQLite() SQLite {
	return SQLite{DSN: defaultSQLiteDSN, MaxBytes: defaultMaxDatabaseBytes}
}

func defaultPage() Page {
	return Page{MaxPages: defaultMaxPages}
}

func defaultStorage() Storage {
	return Storage{SQLite: defaultSQLite(), Page: defaultPage()}
}

func loadSQLite(sqlite *SQLite) error {
	loadString(&sqlite.DSN, envSQLiteDSN)

	return parseInt64(&sqlite.MaxBytes, envSQLiteMaxBytes)
}

func loadPage(page *Page) error {
	return parseInt64(&page.MaxPages, envPageMaxPages)
}

func validateStorage(storage Storage) error {
	if strings.TrimSpace(storage.SQLite.DSN) == "" {
		return invalid(envSQLiteDSN, "must not be empty")
	}

	if storage.SQLite.MaxBytes < 0 || storage.Page.MaxPages < 0 {
		return invalid("storage limits", "must not be negative")
	}

	return nil
}
