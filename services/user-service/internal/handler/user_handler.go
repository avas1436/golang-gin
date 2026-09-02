// services/user-service/internal/handler/grpc_handler.go

package handler

import (
	"context"
	"pkg/grpcerrors"

	pb "pkg/proto/user"
	"user-service/internal/service"
)

// GRPCServer سرور gRPC ما است
type GRPCServer struct {
	pb.UnimplementedUserServiceServer // برای Forward Compatibility
	userService                       *service.UserService
}

// NewGRPCServer یک نمونه جدید از سرور gRPC را می‌سازد
func NewGRPCServer(userService *service.UserService) *GRPCServer {
	return &GRPCServer{
		userService: userService,
	}
}

// Register
func (
	s *GRPCServer,
) Register(
	ctx context.Context,
	req *pb.RegisterRequest,
) (
	*pb.RegisterResponse,
	error,
) {

	resp, err := s.userService.Register(ctx, req)

	if err != nil {
		return nil, grpcerrors.FromAppError(err, "user-service")
	}

	return resp, nil
}

// OTPLogin
func (
	s *GRPCServer,
) Login(
	ctx context.Context,
	req *pb.LoginRequest,
) (
	*pb.LoginResponse,
	error,
) {

	resp, err := s.userService.OTPLogin(ctx, req)

	if err != nil {
		return nil, grpcerrors.FromAppError(err, "user-service")
	}

	return resp, nil
}

// VerifyOTP
func (
	s *GRPCServer,
) VerifyOTP(
	ctx context.Context,
	req *pb.VerifyOTPRequest,
) (
	*pb.VerifyOTPResponse,
	error,
) {

	resp, err := s.userService.VerifyOTP(ctx, req)

	if err != nil {
		return nil, grpcerrors.FromAppError(err, "user-service")
	}

	return resp, nil
}

// RefreshToken
func (
	s *GRPCServer,
) RefreshToken(
	ctx context.Context,
	req *pb.RefreshTokenRequest,
) (
	*pb.RefreshTokenResponse,
	error,
) {

	resp, err := s.userService.RefreshToken(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "user-service")
	}

	return resp, nil
}

// GetUser
func (
	s *GRPCServer,
) GetUser(
	ctx context.Context,
	req *pb.GetUserRequest,
) (
	*pb.User,
	error,
) {

	resp, err := s.userService.GetUser(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "user-service")
	}

	return resp, nil
}

// Logout پیاده‌سازی متد خروج
func (
	s *GRPCServer,
) Logout(
	ctx context.Context,
	req *pb.LogoutRequest,
) (
	*pb.LogoutResponse,
	error,
) {

	resp, err := s.userService.Logout(ctx, req)
	if err != nil {
		return nil, grpcerrors.FromAppError(err, "user-service")
	}

	return resp, nil
}
