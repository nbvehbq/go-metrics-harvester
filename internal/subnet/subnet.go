package subnet

import (
	"context"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

func UnaryServerInterceptor(subnet string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		if subnet != "" {
			var IP string
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				values := md.Get("X-Real-IP")
				if len(values) > 0 {
					IP = values[0]
					realIP := net.ParseIP(IP)
					if realIP == nil {
						return nil, status.Errorf(codes.PermissionDenied, "forbidden")
					}

					_, ipv4Net, err := net.ParseCIDR(subnet)
					if err != nil {
						return nil, status.Errorf(codes.PermissionDenied, "forbidden")
					}

					if !ipv4Net.Contains(realIP) {
						return nil, status.Errorf(codes.PermissionDenied, "forbidden")
					}
				}
			}
		}
		return handler(ctx, req)
	}
}
