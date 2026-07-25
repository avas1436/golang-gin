// services/user-service/internal/service/user_service.go

package service

import (
	"context"
	"errors"
	"fmt"

	"pkg/auth"
	pb "pkg/proto/user"

	"user-service/internal/model"
	"user-service/internal/repository"
)

var ErrInvalidCredentials = errors.New(
	"invalid phone number or password",
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(
	userRepo repository.UserRepository,
) *UserService {

	return &UserService{userRepo: userRepo}
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
