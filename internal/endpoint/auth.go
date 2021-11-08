package endpoint

import (
	"net/http"
	"strings"
)

// BasicAuth is a http.Handler wrapper that handles Basic Authorization.
// It supports only one pair of username and password.
type BasicAuth struct {
	Handler            http.Handler
	Username, Password string
}

// WithBasicAuth wraps http.Handler with a BasicAuth.
func WithBasicAuth(handler http.Handler, userinfo string) http.Handler {
	if userinfo == "" {
		return handler
	}

	a := BasicAuth{Handler: handler}

	xs := strings.SplitN(userinfo, ":", 2)
	a.Username = xs[0]
	if len(xs) > 1 {
		a.Username = xs[0]
		a.Password = xs[1]
	}

	return a
}

func (a BasicAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !ok || username != a.Username || password != a.Password {
		w.Header().Add("WWW-Authenticate", `Basic realm="Ayd? status page"`)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("<h1>Unauthorized</h1>"))
		return
	}

	a.Handler.ServeHTTP(w, r)
}
