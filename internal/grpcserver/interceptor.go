package grpcserver

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"metrics/internal/trustedsubnet"
)

// trustedSubnetKey is the metadata key the agent sends its host IP under.
const trustedSubnetKey = "x-real-ip"

// trustedSubnetInterceptor rejects RPCs whose x-real-ip metadata is missing,
// unparseable, or outside trusted, with codes.PermissionDenied. The decision
// itself lives in trustedsubnet.Checker, shared with the HTTP middleware.
func trustedSubnetInterceptor(trusted *net.IPNet) grpc.UnaryServerInterceptor {
	checker := trustedsubnet.NewChecker(trusted)

	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		rawIP, ok := realIPFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip metadata")
		}

		if !checker.Allow(rawIP) {
			return nil, status.Error(codes.PermissionDenied, "IP address is not in the trusted subnet")
		}

		return handler(ctx, req)
	}
}

// realIPFromContext extracts the client address the agent reported in the
// request metadata.
func realIPFromContext(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get(trustedSubnetKey)
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}
