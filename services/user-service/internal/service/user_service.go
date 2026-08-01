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
)

const otpTTL = 2 * time.Minute

type UserService struct {
	userRepo repository.UserRepository
	otpRepo  repository.OTPRepository
}

func NewUserService(
	userRepo repository.UserRepository,
	otpRepo repository.OTPRepository,
) *UserService {

	return &UserService{userRepo: userRepo, otpRepo: otpRepo}
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

	// TODO(jwt): در قدم بعدی، access_token و refresh_token واقعی
	// اینجا صادر می‌شوند

	return &pb.VerifyOTPResponse{
		User: toProtoUser(user),
	}, nil
}
