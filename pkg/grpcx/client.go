package grpcx

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dial opens a connection to another service with request-ID propagation
// installed. Connections are long-lived: a service dials each dependency once
// at startup and closes it on shutdown.
//
// The transport is insecure because internal traffic is expected to run inside
// a trusted network or behind a service mesh that terminates TLS. Pass
// grpc.WithTransportCredentials to override it.
func Dial(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	base := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(UnaryClientRequestID()),
	}

	conn, err := grpc.NewClient(target, append(base, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", target, err)
	}
	return conn, nil
}
