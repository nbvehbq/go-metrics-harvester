package subnet

import (
	"net"
	"net/http"
)

func WithTructedSubnets(subnet string) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if subnet != "" {
				realIP := net.ParseIP(r.Header.Get("X-Real-IP"))
				if realIP == nil {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}

				_, ipv4Net, err := net.ParseCIDR(subnet)
				if err != nil {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}

				if !ipv4Net.Contains(realIP) {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}

			h.ServeHTTP(w, r)
		}
	}
}
