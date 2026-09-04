// pkg/grpcmiddleware/auth.go
package grpcmiddleware

import (
	"context"
	"strings"

	"pkg/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthInterceptor(
	// کل اینترفیس بررسی توکن را میگیرد
	tokens auth.TokenManager,

	// لیست متد های عمومی و خصوصی
	publicMethods map[string]bool,

) grpc.UnaryServerInterceptor {

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (
		any,
		error,
	) {

		// اگر این روت عمومی بود و نیازی به احراز هویت نداشت رد شو
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// کل متادیتای همراه این درخواست را استخراج میکند
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(
				codes.Unauthenticated,
				"missing request metadata",
			)
		}

		// حالا از متادیتای استخراج شده دنبال authorization میگردیم
		values := md.Get("authorization")
		// اگر خالی بود ارور بده
		if len(values) == 0 {
			return nil, status.Error(
				codes.Unauthenticated,
				"missing authorization token",
			)
		}

		// این فرمت استاندارد ذخیره توکن است
		// Authorization: Bearer <TOKEN>
		// این قسمت از رشته : 'Bearer <TOKEN>'
		// این قسمت رو حذف میکنه : 'Bearer '
		token := strings.TrimPrefix(values[0], "Bearer ")

		// اطلاعات توکن را استخراج میکند و در صورت مشکل ارور میده
		claims, err := tokens.ParseAccessToken(token)
		if err != nil {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid or expired token",
			)
		}

		// از این لحظه به بعد به جای توکن خام اطلاعات کاربر را داریم
		// UserID
		// Role
		ctx = auth.ContextWithClaims(ctx, claims)

		return handler(ctx, req)
	}
}
