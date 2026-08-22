// services/user-service/internal/service/user_service.go

package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pkg/auth"
	pb "pkg/proto/user"

	"user-service/internal/model"
	"user-service/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New(
		"invalid phone number or password",
	)

	ErrInvalidOTP = errors.New(
		"invalid otp code",
	)

	ErrOTPExpiredOrNotFound = errors.New(
		"otp expired or not found, please login again",
	)

	ErrInvalidRefreshToken = errors.New(
		"refresh token is invalid, expired, or already used",
	)
)

const otpTTL = 2 * time.Minute

type UserService struct {
	userRepo         repository.UserRepository
	otpRepo          repository.OTPRepository
	refreshTokenRepo repository.RefreshTokenRepository

	jwtSecret       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewUserService(
	userRepo repository.UserRepository,
	otpRepo repository.OTPRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *UserService {

	return &UserService{
		userRepo:         userRepo,
		otpRepo:          otpRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtSecret:        jwtSecret,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

// issueTokens یک access token (JWT) و یک refresh token (opaque) برای
// کاربر می‌سازد، refresh token را (فقط به‌صورت هش‌شده) در دیتابیس
// ذخیره می‌کند، و نسخه‌ی خام هر دو را برمی‌گرداند تا به کلاینت داده شود.
//
// این تابع را جدا از VerifyOTP و RefreshToken نوشتیم چون هر دو دقیقاً
// همین کار رو نیاز دارن؛ تکرار این منطق در دو جا خطرناکه (مثلاً یادت
// بره در یکیشون expires_at رو درست ست کنی).
func (s *UserService) issueTokens(
	ctx context.Context, user *model.User,
) (
	accessToken string,
	refreshToken string,
	expiresIn int64,
	err error,
) {

	accessToken, err = auth.GenerateAccessToken(
		s.jwtSecret, user.ID, string(user.Role), s.accessTokenTTL,
	)
	if err != nil {
		return "", "", 0, fmt.Errorf(
			"failed to generate access token: %w", err,
		)
	}

	refreshToken, err = auth.GenerateRefreshToken()
	if err != nil {
		return "", "", 0, fmt.Errorf(
			"failed to generate refresh token: %w", err,
		)
	}

	rt := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: auth.HashRefreshToken(refreshToken),
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
	}

	if err := s.refreshTokenRepo.Create(ctx, rt); err != nil {
		return "", "", 0, fmt.Errorf(
			"failed to store refresh token: %w", err,
		)
	}

	return accessToken, refreshToken, int64(s.accessTokenTTL.Seconds()), nil
}

// Register
func (
	s *UserService,
) Register(
	ctx context.Context,
	req *pb.RegisterRequest,
) (
	*pb.RegisterResponse,
	error,
) {

	if req == nil {
		return nil, errors.New("register request is nil")
	}

	hashedPassword, err := auth.HashPassword(req.Password)

	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	newUser := &model.User{
		Email:        req.Email,
		PhoneNumber:  req.PhoneNumber,
		FullName:     req.FullName,
		PasswordHash: hashedPassword,
		Role:         model.RoleMember, // نقش پیش‌فرض برای کاربر جدید
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		if errors.Is(err, repository.ErrDuplicateUser) {
			// لایه handler این خطا رو به کد gRPC مناسب تبدیل می‌کنه
			return nil, err
		}

		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &pb.RegisterResponse{User: toProtoUser(newUser)}, nil
}

// OTP Login
func (
	s *UserService,
) OTPLogin(
	ctx context.Context,
	req *pb.LoginRequest,
) (
	*pb.LoginResponse,
	error,
) {

	user, err := s.userRepo.GetByEmailOrPhone(ctx, req.PhoneNumber)

	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := auth.ComparePassword(
		user.PasswordHash, req.Password,
	); err != nil {

		return nil, ErrInvalidCredentials
	}

	code, err := auth.GenerateOTP()

	if err != nil {
		return nil, fmt.Errorf("failed to generate otp: %w", err)
	}

	challenge := &model.OTPChallenge{
		ID:          uuid.NewString(),
		UserID:      user.ID,
		PhoneNumber: user.PhoneNumber,
		Code:        code,
	}

	if err := s.otpRepo.SaveChallenge(
		ctx, challenge, otpTTL,
	); err != nil {

		return nil, fmt.Errorf(
			"failed to save otp challenge: %w", err,
		)
	}

	// TODO(notification): اینجا باید کد OTP واقعاً برای کاربر پیامک شود.

	return &pb.LoginResponse{
		ChallengeId: challenge.ID,
		RequiresOtp: true,
	}, nil
}

// VerifyOTP
func (s *UserService) VerifyOTP(
	ctx context.Context,
	req *pb.VerifyOTPRequest,
) (
	*pb.VerifyOTPResponse,
	error,
) {

	challenge, err := s.otpRepo.GetChallenge(ctx, req.OtpChallengeId)
	if err != nil {
		if errors.Is(err, repository.ErrOTPChallengeNotFound) {
			return nil, ErrOTPExpiredOrNotFound
		}
		return nil, fmt.Errorf("failed to get otp challenge: %w", err)
	}

	if challenge.Code != req.OtpCode {
		return nil, ErrInvalidOTP
	}

	// کد درست بود؛ چالش حذف می‌شود تا قابل استفاده مجدد (replay) نباشد
	if err := s.otpRepo.DeleteChallenge(
		ctx, req.OtpChallengeId,
	); err != nil {

		return nil, fmt.Errorf(
			"failed to delete otp challenge: %w", err,
		)
	}

	user, err := s.userRepo.GetByID(ctx, challenge.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	accessToken, refreshToken, expiresIn, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &pb.VerifyOTPResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User:         toProtoUser(user),
	}, nil
}

// RefreshToken یک access token جدید (و طبق الگوی rotation، یک refresh
// token جدید) از روی یک refresh token معتبر صادر می‌کند.
func (s *UserService) RefreshToken(
	ctx context.Context,
	req *pb.RefreshTokenRequest,
) (
	*pb.RefreshTokenResponse,
	error,
) {

	tokenHash := auth.HashRefreshToken(req.RefreshToken)

	rt, err := s.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// اگر توکن باطل شده یا منقضی شده، رد می‌شود. توکن باطل‌شده که
	// دوباره استفاده بشه می‌تونه نشونه‌ی سرقت توکن باشه؛ در یک
	// پیاده‌سازی کامل‌تر اینجا جای خوبیه که همه‌ی توکن‌های اون کاربر
	// رو هم باطل کنیم و بهش هشدار بدیم. فعلاً به‌سادگی رد می‌کنیم.
	if rt.Revoked || time.Now().After(rt.ExpiresAt) {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.userRepo.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Rotation: توکن قدیمی باطل می‌شود تا هر بار refresh فقط یک بار
	// قابل استفاده باشد.
	if err := s.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	accessToken, refreshToken, expiresIn, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// GetUser
func (s *UserService) GetUser(
	ctx context.Context,
	req *pb.GetUserRequest,
) (
	*pb.User,
	error,
) {

	user, err := s.userRepo.GetByID(ctx, req.Id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return toProtoUser(user), nil
}

// Logout یک refresh token مشخص را باطل می‌کند.
//
// عمداً idempotent است: اگر توکن از قبل پیدا نشود (مثلاً چون کاربر
// قبلاً logout کرده)، خطا برنمی‌گردانیم — از دید کلاینت، نتیجه‌ی
// نهایی هر دو حالت یکیه: «دیگه لاگین نیستی».
func (s *UserService) Logout(
	ctx context.Context,
	req *pb.LogoutRequest,
) (
	*pb.LogoutResponse,
	error,
) {

	tokenHash := auth.HashRefreshToken(req.RefreshToken)

	rt, err := s.refreshTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return &pb.LogoutResponse{}, nil
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	if err := s.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return &pb.LogoutResponse{}, nil
}
