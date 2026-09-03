// services/user-service/internal/service/user_service.go

package service

import (
	"context"
	"log"
	"pkg/auth"
	appErrors "pkg/errors"
	pb "pkg/proto/user"
	"time"

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

func NewUserService(
	userRepo repository.UserRepository,
	otpRepo repository.OTPRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	tokens auth.TokenManager,
	refreshTokenTTL time.Duration,
) *UserService {

	return &UserService{
		userRepo:         userRepo,
		otpRepo:          otpRepo,
		refreshTokenRepo: refreshTokenRepo,
		tokens:           tokens,
		refreshTokenTTL:  refreshTokenTTL,
	}
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

	accessToken, err := s.tokens.GenerateAccessToken(
		user.ID,
		string(user.Role),
	)
	if err != nil {
		return "", "", 0, err
	}

	refreshToken, err := s.tokens.GenerateRefreshToken()
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
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to get user by phone number or email",
		)
	}

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
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to compare password",
		)
	}

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
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to get user by phone number",
		)
	}

	// تولید OTP
	code, err := auth.GenerateOTP()
	if err != nil {
		return nil, err
	}

	challenge := &model.OTPChallenge{
		ID:          uuid.NewString(),
		UserID:      user.ID,
		PhoneNumber: user.PhoneNumber,
		Code:        code,
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
		ChallengeId:      challenge.ID,
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

	challenge, err := s.otpRepo.GetChallenge(ctx, req.OtpChallengeId)
	if err != nil {

		return nil, err
	}

	if challenge.Code != req.OtpCode {

		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid otp code",
		)

	}

	if err := s.otpRepo.DeleteChallenge(ctx, req.OtpChallengeId); err != nil {
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

	if rt.Revoked || time.Now().After(rt.ExpiresAt) {

		return nil, appErrors.New(
			appErrors.KindNotFound,
			"refresh token is invalid, expired, or already used",
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

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"get user request is nil",
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

	user, err := s.userRepo.GetByID(ctx, req.Id)
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
