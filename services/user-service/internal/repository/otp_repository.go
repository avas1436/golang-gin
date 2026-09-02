// services/user-service/internal/repository/otp_repository.go

package repository

import (
	"context"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	appErrors "pkg/errors"
	"time"
	"user-service/internal/model"

	"github.com/redis/go-redis/v9"
)

type OTPRepository interface {
	SaveChallenge(
		ctx context.Context,
		challenge *model.OTPChallenge,
		ttl time.Duration,
	) error

	GetChallenge(
		ctx context.Context,
		challengeID string,
	) (
		*model.OTPChallenge,
		error,
	)

	DeleteChallenge(
		ctx context.Context,
		challengeID string,
	) error
}

type otpRepository struct {
	client *redis.Client
}

func NewOTPRepository(client *redis.Client) OTPRepository {

	return &otpRepository{client: client}

}

// challengeKey کلید Redis برای ذخیره چالش OTP
func challengeKey(id string) string {

	return fmt.Sprintf("otp_challenge:%s", id)

}

// GetChallenge یک چالش OTP را با ID آن دریافت می‌کند
func (
	r *otpRepository,
) GetChallenge(
	ctx context.Context,
	challengeID string,
) (
	*model.OTPChallenge,
	error,
) {

	// اعتبارسنجی ورودی
	if challengeID == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"challenge id cannot be empty",
		)
	}

	data, err := r.client.Get(ctx, challengeKey(challengeID)).Bytes()

	if err != nil {

		if stdErrors.Is(err, redis.Nil) {

			return nil, appErrors.New(
				appErrors.KindNotFound,
				"OTP challenge not found or expired",
			)

		}

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to get OTP challenge from Redis",
		)

	}

	challenge := &model.OTPChallenge{}

	if err := json.Unmarshal(data, challenge); err != nil {

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to unmarshal OTP challenge",
		)

	}

	return challenge, nil

}

func (
	r *otpRepository,
) SaveChallenge(
	ctx context.Context,
	challenge *model.OTPChallenge,
	ttl time.Duration,
) error {

	// اعتبارسنجی ورودی‌ها
	if challenge == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"challenge cannot be nil",
		)
	}

	if challenge.ID == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"challenge id cannot be empty",
		)
	}

	if challenge.PhoneNumber == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"phone number cannot be empty",
		)
	}

	if challenge.UserID == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"user id cannot be empty",
		)
	}

	if challenge.Code == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"otp code cannot be empty",
		)
	}

	if ttl <= 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"TTL must be greater than zero",
		)
	}

	// مارشال کردن داده
	data, err := json.Marshal(challenge)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to marshal OTP challenge",
		)
	}

	// ذخیره در ردیس
	if err := r.client.Set(
		ctx,
		challengeKey(challenge.ID),
		data,
		ttl,
	).Err(); err != nil {

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to save OTP challenge in Redis",
		)
	}

	return nil
}

func (
	r *otpRepository,
) DeleteChallenge(
	ctx context.Context,
	challengeID string,
) error {

	// اعتبارسنجی ورودی
	if challengeID == "" {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"challenge id cannot be empty",
		)
	}

	// ابتدا چالش را دریافت می‌کنیم تا UserID را داشته باشیم
	_, err := r.GetChallenge(ctx, challengeID)
	if err != nil {
		// اگر چالش وجود نداشت، نیازی به حذف نیست
		if appErrors.GetKind(err) == appErrors.KindNotFound {
			return nil
		}
		return err
	}

	// حذف چالش
	if err := r.client.Del(ctx, challengeKey(challengeID)).Err(); err != nil {

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to delete OTP challenge from Redis",
		)

	}

	return nil
}
