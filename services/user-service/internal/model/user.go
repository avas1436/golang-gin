// services/user-service/internal/model/user.go

package model

import "time"

type Role string

const (
	RoleAdmin       Role = "admin"
	RoleMember      Role = "member"
	RoleViewer      Role = "viewer"
	RoleUnspecified Role = "unspecified"
)

func (r Role) String() string {
	return string(r)
}

func (r Role) IsValid() bool {
	switch r {

	case RoleAdmin, RoleMember, RoleViewer, RoleUnspecified:
		return true

	default:
		return false

	}
}

// این مدل هرگز به بیرون ارسال نمیشه و پسورد هم در آن هش شده ذخیره میشه و در دیتابیس هم همین مدل ذخیره میشه
type User struct {
	ID           string
	Email        string
	PhoneNumber  string
	FullName     string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// RefreshToken مدل توکن رفرش است که در دیتابیس ذخیره میشه و برای احراز هویت کاربر استفاده میشه
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

// IsValid بررسی می‌کند که آیا توکن فعال و معتبر است یا خیر
func (rt *RefreshToken) IsValid() bool {
	return !rt.Revoked && time.Now().Before(rt.ExpiresAt)
}

// Revoke ابطال دستی توکن
func (rt *RefreshToken) Revoke() {
	rt.Revoked = true
}
