// services/user-service/internal/service/helper.go

package service

import (
	"context"
	"time"
	"user-service/internal/model"
)

const (
	// مدت انقضای کد
	otpTTL = 2 * time.Minute

	// یک هش معتبر bcrypt جهت جلوگیری از Timing Attack
	dummyPasswordHash = "$2a$10$e8T.A0N1E8a1O5r.O4M6e.J9V2O1K1E8a1O5r.O4M6e.J9V2O1K1E"
)

func (
	s *UserService,
) issueTokens(
	ctx context.Context,
	user *model.User,
) (
	accessToken string,
	refreshToken string,
	expiresIn int64,
	err error,
) {

	accessToken, err = s.tokens.GenerateAccessToken(
		user.ID.String(),
		string(user.Role),
	)
	if err != nil {
		return "", "", 0, err
	}

	refreshToken, err = s.tokens.GenerateRefreshToken()
	if err != nil {
		return "", "", 0, err
	}

	rt := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: s.tokens.HashRefreshToken(refreshToken),
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
	}

	if err := s.refreshTokenRepo.Create(ctx, rt); err != nil {
		return "", "", 0, err
	}

	return accessToken,
		refreshToken,
		int64(s.tokens.AccessTokenTTL().Seconds()),
		nil
}
