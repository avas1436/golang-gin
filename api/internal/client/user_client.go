// api/internal/client/user_client.go

package client

import (
	"fmt"

	pb "pkg/proto/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// یک لایه کوچک برای تایپ اتصال به سرویس
type UserClient struct {
	pb.UserServiceClient

	conn *grpc.ClientConn
}

// NewUserClient یک اتصال gRPC به user-service برقرار می‌کند.
func NewUserClient(addr string) (*UserClient, error) {

	// یک اتصال غیر ایمن برای وصل شدن به سرویس
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to dial user-service: %w", err)
	}

	return &UserClient{
		UserServiceClient: pb.NewUserServiceClient(conn),
		conn:              conn,
	}, nil
}

// قطع کردن اتصال به سرویس
func (c *UserClient) Close() error {
	return c.conn.Close()
}
