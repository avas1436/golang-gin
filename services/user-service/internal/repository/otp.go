// services/user-service/internal/repository/otp.go

package repository

import (
	"context"
	"encoding/json"
	stdErrors "errors"
	"pkg/cache"
	appErrors "pkg/errors"
	"time"
	"user-service/internal/model"

	"github.com/google/uuid"
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
		challengeID uuid.UUID,
	) (
		*model.OTPChallenge,
		error,
	)

	DeleteChallenge(
		ctx context.Context,
		challengeID uuid.UUID,
	) error
}

type otpRepository struct {
	client     *redis.Client
	keyBuilder cache.KeyBuilder
}

func NewOTPRepository(client *redis.Client) OTPRepository {

	return &otpRepository{
		client:     client,
		keyBuilder: cache.NewKeyBuilder("user-service"),
	}

}

// کلید Redis برای ذخیره چالش OTP
func (r *otpRepository) key(id uuid.UUID) string {

	return r.keyBuilder.Build("otp", id.String())

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

	if challenge.ID == uuid.Nil {
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

	if challenge.UserID == uuid.Nil {
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
		r.key(challenge.ID),
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

// GetChallenge یک چالش OTP را با ID آن دریافت می‌کند
func (
	r *otpRepository,
) GetChallenge(
	ctx context.Context,
	challengeID uuid.UUID,
) (
	*model.OTPChallenge,
	error,
) {

	// اعتبارسنجی ورودی
	if challengeID == uuid.Nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"challenge id cannot be empty",
		)
	}

	// دریافت مقدار از ردیس
	data, err := r.client.Get(ctx, r.key(challengeID)).Bytes()

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
) DeleteChallenge(
	ctx context.Context,
	challengeID uuid.UUID,
) error {

	// اعتبارسنجی ورودی
	if challengeID == uuid.Nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"challenge id cannot be empty",
		)
	}

	// حذف چالش و دریافت تعداد کلید حذف شده
	deletedCount, err := r.client.Del(ctx, r.key(challengeID)).Result()
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to consume otp challenge",
		)
	}

	// اگر تعداد کلید حذف‌شده ۱ نباشد، یعنی کلید قبلاً مصرف شده
	//  یا منقضی شده است
	if deletedCount == 0 {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"otp challenge is invalid, expired, or already consumed",
		)
	}

	return nil
}
