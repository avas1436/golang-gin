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
	accessToken string,
	refreshToken string,
	expiresIn int64,
) *pb.AuthResponse {

	return &pb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User:         toProtoUser(u),
	}

}

// تبدیل اطلاعات کاربر به پروتو
func toProtoUser(u *model.User) *pb.User {

	if u == nil {
		return nil
	}

	return &pb.User{
		Id:          u.ID.String(),
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
