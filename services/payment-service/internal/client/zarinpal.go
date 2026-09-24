// services/payment-service/internal/client/zarinpal.go

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	appErrors "pkg/errors"
)

// zarinpalClient پیاده‌سازی GatewayClient برای زرین‌پال است.
//
// این struct فقط مسئول ارتباط HTTP با API زرین‌پال است.
type zarinpalClient struct {
	merchantID  string
	baseURL     string
	startPayURL string
	httpClient  *http.Client
}

// Structهای داخلی API v4 زرین‌پال
type zarinpalReqPayload struct {
	MerchantID  string            `json:"merchant_id"`
	Amount      int64             `json:"amount"`
	CallbackURL string            `json:"callback_url"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// پاسخ API زرین‌پال برای ایجاد تراکنش است.
type zarinpalReqResponse struct {
	Data struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		Authority string `json:"authority"`
	} `json:"data"`
	Errors []any `json:"errors"`
}

// بدنه درخواست Verify در API زرین‌پال است.
type zarinpalVerifyPayload struct {
	MerchantID string `json:"merchant_id"`
	Amount     int64  `json:"amount"`
	Authority  string `json:"authority"`
}

// پاسخ API زرین‌پال برای Verify است.
type zarinpalVerifyResponse struct {
	Data struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		RefID   int64  `json:"ref_id"`
		CardPan string `json:"card_pan"`
	} `json:"data"`
	Errors []any `json:"errors"`
}

func (
	z *zarinpalClient,
) RequestPayment(
	ctx context.Context,
	input PaymentRequestInput,
) (
	*PaymentRequestOutput,
	error,
) {

	// فقط اطلاعات اختیاری موجود را به metadata اضافه می‌کنیم.
	metadata := make(map[string]string)
	if input.Email != "" {
		metadata["email"] = input.Email
	}
	if input.Mobile != "" {
		metadata["mobile"] = input.Mobile
	}

	// ساخت payload مطابق قرارداد API زرین‌پال.
	payload := zarinpalReqPayload{
		MerchantID:  z.merchantID,
		Amount:      input.Amount,
		CallbackURL: input.CallbackURL,
		Description: input.Description,
		Metadata:    metadata,
	}

	// تبدیل payload به JSON.
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to marshal zarinpal request",
		)
	}

	// endpoint ایجاد تراکنش.
	url := fmt.Sprintf("%s/request.json", z.baseURL)

	// ساخت درخواست HTTP و اتصال context برای timeout/cancellation.
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to create zarinpal request",
		)
	}

	// تنظم هدر های درخواست
	req.Header.Set("Content-Type", "application/json")

	// ارسال درخواست به زرین‌پال.
	resp, err := z.httpClient.Do(req)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"zarinpal gateway request failed",
		)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn(
				"failed to close zarinpal response body",
				"error", err,
			)
		}
	}()

	// خطاهای HTTP مثل 500 یا 429 را قبل از Decode بررسی می‌کنیم.
	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {

		return nil, appErrors.New(
			appErrors.KindInternal,
			fmt.Sprintf(
				"zarinpal returned unexpected HTTP status %d",
				resp.StatusCode,
			),
		)

	}

	// Decode پاسخ JSON زرین‌پال.
	var zResp zarinpalReqResponse

	if err := json.NewDecoder(resp.Body).Decode(&zResp); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to decode zarinpal response",
		)
	}

	// کد ۱۰۰ به معنی تایید اولیه و صدور Authority است
	if zResp.Data.Code != 100 {

		return nil, appErrors.New(
			appErrors.KindInternal,
			fmt.Sprintf(
				"zarinpal error code %d: %s",
				zResp.Data.Code,
				zResp.Data.Message,
			),
		)

	}

	// ساخت URL نهایی برای هدایت کاربر به صفحه پرداخت.
	redirectURL := fmt.Sprintf(
		"%s/%s",
		z.startPayURL,
		zResp.Data.Authority,
	)

	return &PaymentRequestOutput{
		Authority:   zResp.Data.Authority,
		RedirectURL: redirectURL,
	}, nil
}

// نتیجه پرداخت را از زرین‌پال استعلام و تأیید می‌کند.
func (
	z *zarinpalClient,
) VerifyPayment(
	ctx context.Context,
	input PaymentVerifyInput,
) (
	*PaymentVerifyOutput,
	error,
) {

	// ساخت payload مطابق API زرین‌پال.
	payload := zarinpalVerifyPayload{
		MerchantID: z.merchantID,
		Amount:     input.Amount,
		Authority:  input.Authority,
	}

	// تبدیل payload به JSON.
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to marshal zarinpal verify payload",
		)
	}

	// endpoint تأیید تراکنش.
	url := fmt.Sprintf("%s/verify.json", z.baseURL)

	// ساخت درخواست HTTP.
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to create zarinpal verify request",
		)
	}

	// افزودن هدر های درخواست
	req.Header.Set("Content-Type", "application/json")

	// ارسال درخواست Verify.
	resp, err := z.httpClient.Do(req)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"zarinpal verify request failed",
		)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn(
				"failed to close zarinpal response body",
				"error", err,
			)
		}
	}()

	var zResp zarinpalVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&zResp); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to decode zarinpal verify response",
		)
	}

	// بررسی status code مربوط به HTTP.
	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {

		return nil, appErrors.New(
			appErrors.KindInternal,
			fmt.Sprintf(
				"zarinpal returned unexpected HTTP status %d",
				resp.StatusCode,
			),
		)

	}

	// Decode پاسخ زرین‌پال. var zResp zarinpalVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&zResp); err != nil {

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to decode zarinpal verify response",
		)

	}

	// کد 100 یعنی پرداخت با موفقیت تأیید شده است.
	//
	// کد 101 یعنی پرداخت قبلاً Verify شده است.
	//  بنابراین هر دو حالت را موفق در نظر می‌گیریم.
	if zResp.Data.Code != 100 && zResp.Data.Code != 101 {

		return &PaymentVerifyOutput{
				Success: false,
			}, appErrors.New(
				appErrors.KindInvalidInput,
				fmt.Sprintf(
					"payment verification failed with code %d: %s",
					zResp.Data.Code,
					zResp.Data.Message,
				),
			)
	}

	return &PaymentVerifyOutput{
		RefID:   fmt.Sprintf("%d", zResp.Data.RefID),
		Success: true,
		CardPan: zResp.Data.CardPan,
	}, nil
}
