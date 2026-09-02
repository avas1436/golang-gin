// services/user-service/internal/service/mapper.go

package service

import (
	pb "pkg/proto/user"
	"user-service/internal/model"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ساخت پاسخ احراز هویت
func toProtoAuth(
	u *model.User,
	access_token string,
	refresh_token string,
	expires_in int64,
) *pb.AuthResponse {

	return &pb.AuthResponse{
		AccessToken:  access_token,
		RefreshToken: refresh_token,
		ExpiresIn:    expires_in,
		User:         toProtoUser(u),
	}
}

// تبدیل اطلاعات کاربر به پروتو
func toProtoUser(u *model.User) *pb.User {
	return &pb.User{
		Id:          u.ID,
		Email:       u.Email,
		PhoneNumber: u.PhoneNumber,
		FullName:    u.FullName,
		Role:        toProtoRole(u.Role),
		CreatedAt:   timestamppb.New(u.CreatedAt),
	}
}

// تبدیل نقش کاربر به پروتو
func toProtoRole(r model.Role) pb.Role {
	switch r {
	case model.RoleAdmin:
		return pb.Role_ROLE_ADMIN

	case model.RoleMember:
		return pb.Role_ROLE_MEMBER

	case model.RoleViewer:
		return pb.Role_ROLE_VIEWER

	default:
		return pb.Role_ROLE_UNSPECIFIED
	}
}
