// services/user-service/internal/service/user_service.go

package service

import (
	"context"
	"log"
	"time"

	"pkg/auth"
	appErrors "pkg/errors"
	pb "pkg/proto/user"
	"user-service/internal/model"
	"user-service/internal/repository"

	"github.com/google/uuid"
)

const otpTTL = 2 * time.Minute

type UserService struct {
	userRepo         repository.UserRepository
	otpRepo          repository.OTPRepository
	refreshTokenRepo repository.RefreshTokenRepository

	tokens          auth.TokenManager
	refreshTokenTTL time.Duration
}

func (
	s *UserService,
) issueTokens(
	ctx context.Context,
	user *model.User,
) (
	access_token string,
	refresh_token string,
	expires_in int64,
	err error,
) {

	// start := time.Now()

	accessToken, err := s.tokens.GenerateAccessToken(
		user.ID.String(),
		string(user.Role),
	)
	if err != nil {
		return "", "", 0, err
	}

	// log.Printf("GenerateAccessToken: %s", time.Since(start))

	// start = time.Now()

	refreshToken, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return "", "", 0, err
	}

	// log.Printf("CreateRefreshToken: %s", time.Since(start))

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

// Register
func (
	s *UserService,
) Register(
	ctx context.Context,
	req *pb.RegisterRequest,
) (
	*pb.AuthResponse,
	error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"register request is nil",
		)
	}

	if req.Email == "" || req.PhoneNumber == "" || req.Password == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"email, phone number, and password are required",
		)
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	newUser := &model.User{
		Email:        req.Email,
		PhoneNumber:  req.PhoneNumber,
		FullName:     req.FullName,
		PasswordHash: hashedPassword,
		Role:         model.RoleMember,
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {

		return nil, err

	}

	accessToken, refreshToken, expireIn, err := s.issueTokens(
		ctx,
		newUser,
	)
	if err != nil {
		return nil, err
	}

	return toProtoAuth(
			newUser,
			accessToken,
			refreshToken,
			expireIn,
		),
		nil
}

// Password Login
func (
	s *UserService,
) PasswordLogin(
	ctx context.Context,
	req *pb.PasswordLoginRequest,
) (
	*pb.AuthResponse,
	error,
) {

	// start := time.Now()

	//  اعتبارسنجی ورودی
	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"login request cannot be nil",
		)
	}

	if req.Identifier == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"phone number or email address is required",
		)
	}

	if req.Password == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"password is required",
		)
	}

	// log.Printf("Validation: %s", time.Since(start))

	// start = time.Now()

	user, err := s.userRepo.GetByEmailOrPhone(ctx, req.Identifier)
	if err != nil {

		if appErrors.GetKind(err) == appErrors.KindNotFound {

			// برای امنیت، پیام یکسان می‌دهیم
			return nil, appErrors.New(
				appErrors.KindUnauthenticated,
				"invalid phone number or email",
			)

		}

		// خطاهای داخلی
		return nil, err
	}

	// log.Printf("GetUser: %s", time.Since(start))

	// start = time.Now()

	// مقایسه رمز عبور
	if err := auth.ComparePassword(
		user.PasswordHash,
		req.Password,
	); err != nil {

		if appErrors.GetKind(err) == appErrors.KindInvalidInput {

			return nil, appErrors.New(
				appErrors.KindUnauthenticated,
				"invalid phone number or password",
			)

		}

		// خطاهای داخلی در مقایسه رمز
		return nil, err
	}

	// log.Printf("ComparePassword: %s", time.Since(start))

	accessToken, refreshToken, expireIn, err := s.issueTokens(
		ctx,
		user,
	)
	if err != nil {
		return nil, err
	}

	return toProtoAuth(
			user,
			accessToken,
			refreshToken,
			expireIn,
		),
		nil
}

// OTP Login
func (
	s *UserService,
) OTPLogin(
	ctx context.Context,
	req *pb.OTPLoginRequest,
) (
	*pb.OTPLoginResponse,
	error,
) {

	//  اعتبارسنجی ورودی
	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"login request cannot be nil",
		)
	}

	if req.PhoneNumber == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"phone number is required",
		)
	}

	user, err := s.userRepo.GetByEmailOrPhone(ctx, req.PhoneNumber)
	if err != nil {

		if appErrors.GetKind(err) == appErrors.KindNotFound {

			// برای امنیت، پیام یکسان می‌دهیم
			return nil, appErrors.New(
				appErrors.KindUnauthenticated,
				"invalid phone number or email",
			)

		}

		// خطاهای داخلی
		return nil, err
	}

	// تولید OTP
	code, err := auth.GenerateOTP()
	if err != nil {
		return nil, err
	}

	challenge := &model.OTPChallenge{
		ID:          uuid.New(),
		UserID:      user.ID,
		PhoneNumber: user.PhoneNumber,
		Code:        code,
		ExpiresAt:   time.Now().UTC().Add(otpTTL),
	}

	if err := s.otpRepo.SaveChallenge(
		ctx,
		challenge,
		otpTTL,
	); err != nil {

		return nil, err
	}

	// TODO(notification): اینجا باید کد OTP واقعاً برای کاربر پیامک شود.
	log.Printf("user OTP Code is :%s", code)

	return &pb.OTPLoginResponse{
		ChallengeId:      challenge.ID.String(),
		ExpiresInSeconds: int32(otpTTL.Seconds()),
	}, nil
}

// VerifyOTP
func (
	s *UserService,
) VerifyOTP(
	ctx context.Context,
	req *pb.VerifyOTPRequest,
) (
	*pb.AuthResponse,
	error,
) {

	// اعتبار سنجی داده ورودی
	if req == nil || req.OtpChallengeId == "" || req.OtpCode == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"challenge id and code are required",
		)
	}

	// تبدیل رشته آیدی به فرمت uuid
	challengeID, err := uuid.Parse(req.OtpChallengeId)
	if err != nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid challenge id format",
		)
	}

	challenge, err := s.otpRepo.GetChallenge(ctx, challengeID)
	if err != nil {

		return nil, err
	}

	if challenge.Code != req.OtpCode {

		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid otp code",
		)

	}

	if err := s.otpRepo.DeleteChallenge(ctx, challengeID); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, challenge.UserID)
	if err != nil {
		return nil, err
	}

	accessToken, refreshToken, expiresIn, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return toProtoAuth(
			user,
			accessToken,
			refreshToken,
			expiresIn,
		),
		nil
}

// RefreshToken
func (
	s *UserService,
) RefreshToken(
	ctx context.Context,
	req *pb.RefreshTokenRequest,
) (
	*pb.AuthResponse,
	error,
) {

	// اعتبار سنجی
	if req == nil || req.RefreshToken == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"refresh token is required",
		)
	}

	tokenHash := s.tokens.HashRefreshToken(req.RefreshToken)

	rt, err := s.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {

		if appErrors.GetKind(err) == appErrors.KindNotFound {
			return nil, appErrors.New(
				appErrors.KindNotFound,
				"refresh token is invalid, expired, or already used",
			)
		}

		return nil, err
	}

	// استفاده از متد ساختار رفرش توکن برای اعتبار سنجی آن
	if !rt.IsValid() {
		return nil, appErrors.New(
			appErrors.KindUnauthenticated,
			"refresh token is invalid, expired, or revoked",
		)
	}

	user, err := s.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, err
	}

	if err := s.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
		return nil, err
	}

	accessToken, refreshToken, expiresIn, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return toProtoAuth(
			user,
			accessToken,
			refreshToken,
			expiresIn,
		),
		nil
}

// GetUser
func (
	s *UserService,
) GetUser(
	ctx context.Context,
	req *pb.GetUserRequest,
) (
	*pb.User,
	error,
) {

	if req == nil || req.Id == "" {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"user id is required",
		)
	}

	targetUserID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid user id format",
		)
	}

	// ابتدا مقادیر احراز هویت استخراج میشه و اگه مشکلی داشت ارور
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, appErrors.New(
			appErrors.KindUnauthenticated,
			"authentication required",
		)
	}

	// در این قسمت RBAC کنترل میشود
	// تنها خود کاربران و ادمین ها میتوانند دسترسی پیدا کنند
	if claims.UserID != req.Id && claims.Role != string(model.RoleAdmin) {
		return nil, appErrors.New(
			appErrors.KindPermissionDenied,
			"you are not allowed to view this user's profile",
		)
	}

	user, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {

		return nil, err

	}

	return toProtoUser(user), nil
}

// Logout
func (
	s *UserService,
) Logout(
	ctx context.Context,
	req *pb.LogoutRequest,
) (
	*pb.LogoutResponse,
	error,
) {

	if req == nil || req.RefreshToken == "" {
		return &pb.LogoutResponse{}, nil
	}

	tokenHash := s.tokens.HashRefreshToken(req.RefreshToken)

	rt, err := s.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {

		if appErrors.GetKind(err) == appErrors.KindNotFound {
			return &pb.LogoutResponse{}, nil
		}

		return nil, err
	}

	if err := s.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
		return nil, err
	}

	return &pb.LogoutResponse{}, nil
}
