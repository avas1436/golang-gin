// services/payment-service/internal/client/zarinpal.go

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	appErrors "pkg/errors"
)

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

type zarinpalReqResponse struct {
	Data struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		Authority string `json:"authority"`
	} `json:"data"`
	Errors []any `json:"errors"`
}

type zarinpalVerifyPayload struct {
	MerchantID string `json:"merchant_id"`
	Amount     int64  `json:"amount"`
	Authority  string `json:"authority"`
}

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

	metadata := make(map[string]string)
	if input.Email != "" {
		metadata["email"] = input.Email
	}
	if input.Mobile != "" {
		metadata["mobile"] = input.Mobile
	}

	payload := zarinpalReqPayload{
		MerchantID:  z.merchantID,
		Amount:      input.Amount,
		CallbackURL: input.CallbackURL,
		Description: input.Description,
		Metadata:    metadata,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to marshal zarinpal request",
		)
	}

	url := fmt.Sprintf("%s/request.json", z.baseURL)
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
	req.Header.Set("Content-Type", "application/json")

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"zarinpal gateway request failed",
		)
	}
	defer resp.Body.Close()

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

	redirectURL := fmt.Sprintf("%s/%s", z.startPayURL, zResp.Data.Authority)

	return &PaymentRequestOutput{
		Authority:   zResp.Data.Authority,
		RedirectURL: redirectURL,
	}, nil
}

func (
	z *zarinpalClient,
) VerifyPayment(
	ctx context.Context,
	input PaymentVerifyInput,
) (
	*PaymentVerifyOutput,
	error,
) {

	payload := zarinpalVerifyPayload{
		MerchantID: z.merchantID,
		Amount:     input.Amount,
		Authority:  input.Authority,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to marshal zarinpal verify payload",
		)
	}

	url := fmt.Sprintf("%s/verify.json", z.baseURL)
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
	req.Header.Set("Content-Type", "application/json")

	resp, err := z.httpClient.Do(req)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"zarinpal verify request failed",
		)
	}
	defer resp.Body.Close()

	var zResp zarinpalVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&zResp); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to decode zarinpal verify response",
		)
	}

	// کد ۱۰۰ یعنی پرداخت با موفقیت انجام شد، کد ۱۰۱ یعنی قبلاً Verify شده است
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
