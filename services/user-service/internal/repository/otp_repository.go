// services/user-service/internal/repository/otp_repository.go

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"user-service/internal/model"

	"github.com/redis/go-redis/v9"
)

var ErrOTPChallengeNotFound = errors.New("otp challenge not found or expired")

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

func challengeKey(id string) string {

	return fmt.Sprintf("otp_challenge:%s", id)

}

func (r *otpRepository) GetChallenge(
	ctx context.Context,
	challengeID string,
) (
	*model.OTPChallenge,
	error,
) {

	data, err := r.client.Get(ctx, challengeKey(challengeID)).Bytes()

	if errors.Is(err, redis.Nil) {
		return nil, ErrOTPChallengeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get otp challenge: %w", err)
	}

	challenge := &model.OTPChallenge{}

	if err := json.Unmarshal(data, challenge); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal otp challenge: %w", err,
		)
	}

	return challenge, nil

}

func (r *otpRepository) SaveChallenge(
	ctx context.Context,
	challenge *model.OTPChallenge,
	ttl time.Duration,
) error {

	data, err := json.Marshal(challenge)
	if err != nil {
		return fmt.Errorf("failed to marshal otp challenge: %w", err)
	}

	if err := r.client.Set(
		ctx,
		challengeKey(challenge.ID),
		data,
		ttl,
	).Err(); err != nil {

		return fmt.Errorf("failed to save otp challenge: %w", err)
	}

	return nil
}

func (r *otpRepository) DeleteChallenge(
	ctx context.Context,
	challengeID string,
) error {

	if err := r.client.Del(
		ctx,
		challengeKey(challengeID),
	).Err(); err != nil {

		return fmt.Errorf("failed to delete otp challenge: %w", err)
	}

	return nil
}
