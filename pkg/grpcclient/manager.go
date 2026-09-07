// pkg/grpcclient/manager.go

package grpcclient

// دلایل استفاده از این لایه:
// مدیریت متمرکز تنظیمات: اگر روزی بخواهیم تنظیماتی مانند Tracing، Retry یا Timeout را تغییر دهیم، تنها کافی است این لایه را تغییر دهیم و همه سرویس‌ها از آن بهره‌مند خواهند شد.
// جلوگیری از نشت حافظه و منابع: با مدیریت متمرکز اتصالات، می‌توانیم اطمینان حاصل کنیم که اتصالات به درستی بسته می‌شوند و منابع سیستم آزاد می‌شوند.
// تجرید لایه شبکه: این لایه به عنوان یک واسط بین کدهای سرویس و gRPC عمل می‌کند و پیچیدگی‌های مربوط به مدیریت اتصالات را از کدهای اصلی سرویس جدا می‌کند.

import (
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// این ساختار یک اتصال سنگین میسازد که نباید در زمان اجرای برنامه
//
//	چندین نسخه از آن داشته باشیم بلکه یک اتصال را در طول عمر برنامه
//
// نگه می‌داریم و از آن استفاده می‌کنیم. این باعث کاهش سربار
//
//	و افزایش کارایی می‌شود.
//
// grpc.ClientConn
type ClientManager struct {
	conns []*grpc.ClientConn
}

// الگوی Singleton برای مدیریت اتصالات gRPC
// یعنی یک نمونه از ClientManager در طول عمر برنامه وجود دارد و همه
func NewManager() *ClientManager {
	return &ClientManager{conns: make([]*grpc.ClientConn, 0)}
}

// Dial یک اتصال جدید با تنظیمات یکسان (Tracing, Retry, Timeout) می‌سازد
func (
	m *ClientManager,
) Dial(
	addr string,
	opts ...grpc.DialOption,
) (
	*grpc.ClientConn,
	error,
) {

	defaultOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// محل اضافه کردن Interceptor ها برای Tracing و Retry و Timeout
	}

	// اضافه کردن گزینه‌های اضافی که کاربر می‌تواند
	//  برای سفارشی‌سازی اتصال ارائه دهد
	defaultOpts = append(defaultOpts, opts...)

	conn, err := grpc.NewClient(addr, defaultOpts...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	m.conns = append(m.conns, conn)
	return conn, nil
}

// CloseAll تمامی اتصالات ساخته‌شده را یکجا می‌بندد
func (m *ClientManager) CloseAll() error {
	var errs []error
	for _, conn := range m.conns {
		if err := conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
