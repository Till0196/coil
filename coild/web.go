package coild

import (
	"net/http"
	"strings"
)

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/status" {
		s.handleStatus(w, r)
		return
	}

	if r.URL.Path == "/ip" {
		s.handleNewIP(w, r)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/ip/") {
		renderError(r.Context(), w, APIErrNotFound)
		return
	}

	podKey := r.URL.Path[4:]
	if len(podKey) == 0 {
		renderError(r.Context(), w, APIErrBadRequest)
		return
	}
	keys := strings.SplitN(podKey, "/", 3)
	if len(keys) != 3 {
		renderError(r.Context(), w, APIErrBadRequest)
		return
	}
	if keys[0] == "" || keys[1] == "" || keys[2] == "" {
		renderError(r.Context(), w, APIErrBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleIPGet(w, r, keys[2])
	case http.MethodDelete:
		s.handleIPDelete(w, r, keys)
	default:
		renderError(r.Context(), w, APIErrBadMethod)
	}
}
