// services/user-service/internal/handler/routes.go

package handler

import (
	"pkg/grpcmiddleware"
	pb "pkg/proto/user"
	"pkg/ratelimit"
)

// این ها لیست متد های عمومی است
// هر متدی در این لیست نباشد در صورت فراخانی false میدهد
func PublicMethods() map[string]bool {

	return map[string]bool{
		pb.UserService_Register_FullMethodName:      true,
		pb.UserService_PasswordLogin_FullMethodName: true,
		pb.UserService_OTPLogin_FullMethodName:      true,
		pb.UserService_VerifyOTP_FullMethodName:     true,
		pb.UserService_RefreshToken_FullMethodName:  true,
	}
}

// لیست متد ها به همراه محدودیت ها
func RateLimitRules() map[string]grpcmiddleware.RateLimitRule {

	return map[string]grpcmiddleware.RateLimitRule{
		pb.UserService_Register_FullMethodName: {
			Limit: ratelimit.PerMinute(5),
		},

		pb.UserService_PasswordLogin_FullMethodName: {
			Limit: ratelimit.PerMinute(5),
		},

		pb.UserService_OTPLogin_FullMethodName: {
			Limit: ratelimit.PerMinute(3),
		},

		pb.UserService_VerifyOTP_FullMethodName: {
			Limit: ratelimit.PerMinute(5),
		},

		pb.UserService_RefreshToken_FullMethodName: {
			Limit: ratelimit.PerMinute(10),
		},

		pb.UserService_GetUser_FullMethodName: {
			Limit: ratelimit.PerMinute(60),
		},

		pb.UserService_Logout_FullMethodName: {
			Limit: ratelimit.PerMinute(10),
		},
	}
}
