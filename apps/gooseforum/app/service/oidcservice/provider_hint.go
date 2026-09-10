package oidcservice

import (
	"net/http"
	"net/url"
)

// withProviderHint selects an existing social login after the provider has
// validated and persisted the authorization request. It does not authenticate,
// change an already-authenticated redirect, or bypass the browser binding.
// Only the server-generated login bridge can be rewritten; arbitrary hints and
// destinations keep the standard OIDC flow.
func withProviderHint(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hint := r.URL.Query().Get("login_hint")
		if hint == "google" || hint == "github" {
			w = &providerHintWriter{ResponseWriter: w, provider: hint}
		}
		next.ServeHTTP(w, r)
	})
}

type providerHintWriter struct {
	http.ResponseWriter
	provider string
}

func (w *providerHintWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *providerHintWriter) WriteHeader(status int) {
	if status == http.StatusFound || status == http.StatusSeeOther {
		login, err := url.Parse(w.Header().Get("Location"))
		if err == nil && login.Scheme == "" && login.Host == "" && login.Path == "/login" {
			callback := login.Query().Get("redirect")
			target, err := url.Parse(callback)
			if err == nil && target.Scheme == "" && target.Host == "" &&
				target.Path == issuerPath+"/authorize/callback" && target.Query().Get("id") != "" {
				w.Header().Set("Location", "/api/auth/"+w.provider+"?redirect="+url.QueryEscape(callback))
			}
		}
	}
	w.ResponseWriter.WriteHeader(status)
}
