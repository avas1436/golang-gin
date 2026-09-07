// api/internal/client/client.go

package client

import (
	"fmt"

	"pkg/grpcclient"
	pbUser "pkg/proto/user"
)

// تمامی سرویس هایی که اضافه میشن میان اینجا
type Config struct {
	UserServiceAddr string
	// OrderServiceAddr string
}

// تمام کلاینت‌های آماده‌ی protobuf پروژه است
type Clients struct {
	User pbUser.UserServiceClient
	// Order pbOrder.UserServiceClient

	manager *grpcclient.ClientManager
}

// New تمام اتصالات gRPC را برقرار کرده و کلاینت‌ها را تحویل می‌دهد
func New(cfg Config) (*Clients, error) {

	// ایجاد یک مدیر اتصال gRPC
	manager := grpcclient.NewManager()

	// 1. User Service Client
	// این کلاینت تنها اینترسپتر های پیش فرض را دارد
	userConn, err := manager.Dial(cfg.UserServiceAddr)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to connect to user-service: %w",
			err,
		)
	}

	// 2. Order Service Client
	// orderConn, err := manager.Dial(cfg.OrderServiceAddr)
	// if err != nil {
	// 	_ = manager.CloseAll()
	// 	return nil, fmt.Errorf("connect order-service: %w", err)
	// }

	return &Clients{
		User:    pbUser.NewUserServiceClient(userConn),
		manager: manager,
	}, nil
}

// قطع کردن اتصال به سرویس
func (c *Clients) Close() error {
	if c.manager == nil {
		return nil
	}
	return c.manager.CloseAll()
}
