package httpapi

import "net/http"

func requestRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		recordRequestRoute(writer, request.Pattern)
		next.ServeHTTP(writer, request)
	})
}

func recordRequestRoute(writer http.ResponseWriter, pattern string) {
	for {
		recorder, ok := writer.(interface{ RecordRoute(pattern string) })
		if ok {
			recorder.RecordRoute(pattern)
		}

		unwrapper, ok := writer.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return
		}

		writer = unwrapper.Unwrap()
	}
}
