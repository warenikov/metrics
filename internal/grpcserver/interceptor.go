package grpcserver

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// trustedSubnetKey is the metadata key the agent sends its host IP under.
const trustedSubnetKey = "x-real-ip"

// trustedSubnetInterceptor rejects RPCs whose x-real-ip metadata is missing,
// unparseable, or outside trusted, with codes.PermissionDenied.
func trustedSubnetInterceptor(trusted *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip metadata")
		}

		values := md.Get(trustedSubnetKey)
		if len(values) == 0 {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip metadata")
		}

		ip := net.ParseIP(values[0])
		if ip == nil || !trusted.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "IP address is not in the trusted subnet")
		}

		return handler(ctx, req)
	}
}
