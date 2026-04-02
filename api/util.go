package main

import "net/http"

type statusInterceptor struct {
	http.ResponseWriter
}

func (w *statusInterceptor) WriteHeader(code int) {
	if code == http.StatusNoContent {
		w.ResponseWriter.WriteHeader(http.StatusOK)
		return
	}
	w.ResponseWriter.WriteHeader(code)
}

func intercept204(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&statusInterceptor{ResponseWriter: w}, r)
	})
}
