package client_ip_middleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/samber/lo"
)

var xffKey = http.CanonicalHeaderKey("X-Forwarded-For")

type clientIPContextKey struct{}

func FromContext(ctx context.Context) string {
	return ctx.Value(clientIPContextKey{}).(string)
}

func ClientIP(h http.Handler) http.Handler {
	var fn = func(w http.ResponseWriter, r *http.Request) {
		var clientIp = parseXff(r.Header.Get(xffKey))

		if len(clientIp) == 0 {
			clientIp = r.RemoteAddr
		}

		clientIp, _ = splitHostPort(clientIp)
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPContextKey{}, clientIp)))
	}

	return http.HandlerFunc(fn)
}

func parseXff(v string) string {
	if values := lo.Compact(strings.Split(v, ",")); len(values) == 0 {
		return ""
	} else {
		return values[len(values)-1]
	}
}

func splitHostPort(s string) (string, string) {
	s = strings.Trim(s, " ")

	h, p, err := net.SplitHostPort(s)

	if err != nil {
		return s, ""
	}

	return h, p
}
