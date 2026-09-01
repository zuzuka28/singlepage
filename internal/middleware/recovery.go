package middleware

import (
	"fmt"
	"log"
	"net/http"
)

// Recovery converts panics to safe HTTP 500 responses and logs evidence.
func Recovery(next http.Handler) http.Handler {
	return RecoveryWithLogFunc(defaultPanicLogf)(next)
}

func defaultPanicLogf(format string, args ...any) {
	err := log.Output(2, fmt.Sprintf(format, args...))
	if err != nil {
		return
	}
}
