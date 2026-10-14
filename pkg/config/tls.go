// pkg/config/tls.go

package config

// TLSConfig تنظیمات مربوط به گواهی‌ها و ارتباط امن mTLS را نگهداری می‌کند
type TLSConfig struct {
	CAFile   string `mapstructure:"ca_file" yaml:"ca_file"`
	CertFile string `mapstructure:"cert_file" yaml:"cert_file"`
	KeyFile  string `mapstructure:"key_file" yaml:"key_file"`
}
