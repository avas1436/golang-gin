// services/user-service/internal/service/mapper.go

package service

import (
	pb "pkg/proto/user"
	"user-service/internal/model"

	"google.golang.org/protobuf/types/known/timestamppb"
)

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
