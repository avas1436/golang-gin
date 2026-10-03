// pkg/config/worker.go

package config

import "time"

// WorkerConfig تنظیمات عمومی مربوط به Background Workerها را نگهداری می‌کند
type WorkerConfig struct {
	Interval     time.Duration `mapstructure:"interval" yaml:"interval"`
	StaleTimeout time.Duration `mapstructure:"stale_timeout" yaml:"stale_timeout"`
}
