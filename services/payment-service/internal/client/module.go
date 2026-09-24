// services/payment-service/internal/client/module.go

package client

import (
	"net/http"
	"payment-service/config"
	commonConfig "pkg/config"
	"time"

	"go.uber.org/fx"
)

// ایجاد کانفیگ زرین پال
func NewZarinpalConfig(cfg *config.Config) commonConfig.ZarinpalConfig {

	return commonConfig.ZarinpalConfig{
		MerchantID: cfg.ZarinPal.MerchantID,
		IsSandbox:  cfg.ZarinPal.IsSandbox,
	}

}

// ایجاد کلاینت زرین پال
func NewZarinpalClient(cfg commonConfig.ZarinpalConfig) GatewayClient {
	baseURL := "https://api.zarinpal.com/pg/v4/payment"
	startPayURL := "https://www.zarinpal.com/pg/StartPay"

	// در محیط توسعه می‌توانیم از sandbox زرین‌پال استفاده کنیم.
	if cfg.IsSandbox {
		baseURL = "https://sandbox.zarinpal.com/pg/v4/payment"
		startPayURL = "https://sandbox.zarinpal.com/pg/StartPay"
	}

	return &zarinpalClient{
		merchantID:  cfg.MerchantID,
		baseURL:     baseURL,
		startPayURL: startPayURL,

		// HTTP client مشترک این adapter.
		// Timeout از معطل ماندن دائمی درخواست جلوگیری می‌کند.
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

var Module = fx.Module(
	"client",
	fx.Provide(
		NewZarinpalConfig,
		NewZarinpalClient,
	),
)
