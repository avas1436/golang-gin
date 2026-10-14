// pkg/grpcclient/manager.go

package grpcclient

// دلایل استفاده از این لایه:
// مدیریت متمرکز تنظیمات: اگر روزی بخواهیم تنظیماتی مانند Tracing، Retry یا Timeout را تغییر دهیم، تنها کافی است این لایه را تغییر دهیم و همه سرویس‌ها از آن بهره‌مند خواهند شد.
// جلوگیری از نشت حافظه و منابع: با مدیریت متمرکز اتصالات، می‌توانیم اطمینان حاصل کنیم که اتصالات به درستی بسته می‌شوند و منابع سیستم آزاد می‌شوند.
// تجرید لایه شبکه: این لایه به عنوان یک واسط بین کدهای سرویس و gRPC عمل می‌کند و پیچیدگی‌های مربوط به مدیریت اتصالات را از کدهای اصلی سرویس جدا می‌کند.

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
// مدیریت‌کننده متمرکز اتصالات gRPC با قابلیت کش کردن و ایمنی در برابر همگامی
type ClientManager struct {
	mu sync.Mutex
	// نگهداری اتصالات به ازای آدرس برای جلوگیری از ایجاد کانکشن‌های تکراری
	conns  map[string]*grpc.ClientConn
	tlsCfg *tls.Config
}

// الگوی Singleton برای مدیریت اتصالات gRPC
// یعنی یک نمونه از ClientManager در طول عمر برنامه وجود دارد و همه
func NewManager(tlsCfg *tls.Config) *ClientManager {
	return &ClientManager{
		conns:  make(map[string]*grpc.ClientConn),
		tlsCfg: tlsCfg,
	}
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

	m.mu.Lock()
	defer m.mu.Unlock()

	// اگر اتصال از قبل برای این آدرس وجود دارد، همان را برگردان
	// (جلوگیری از هدر رفت منابع)
	if conn, exists := m.conns[addr]; exists {
		return conn, nil
	}

	var creds credentials.TransportCredentials

	if m.tlsCfg != nil {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("bad addr %s: %w", addr, err)
		}
		cfg := m.tlsCfg.Clone()
		cfg.ServerName = host // باید با SAN گواهی سرویس یکی باشد
		creds = credentials.NewTLS(cfg)
	} else {
		creds = insecure.NewCredentials() // فقط وقتی TLS_ENABLED=false
	}

	all := append(
		[]grpc.DialOption{grpc.WithTransportCredentials(creds)},
		opts...,
	)

	conn, err := grpc.NewClient(addr, all...)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	m.conns[addr] = conn

	return conn, nil
}

// CloseAll تمامی اتصالات ساخته‌شده را یکجا می‌بندد
func (m *ClientManager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for addr, conn := range m.conns {
		if err := conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close conn %s: %w", addr, err))
		}
	}
	// پاکسازی مپ پس از بسته شدن اتصالات
	m.conns = make(map[string]*grpc.ClientConn)

	return errors.Join(errs...)
}
