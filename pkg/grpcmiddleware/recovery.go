// pkg/grpcmiddleware/recovery.go
package grpcmiddleware

// کلیت کار این فایل این است که اگر یک جایی پنیک رخ داد کل برنامه نخوابه

import (
	"context"
	"log"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpc.UnaryServerInterceptor
// این دیتا تابپ میگوید من یک تابع مانند تابع بدون نام داخلی ما میخواهد

// grpc.UnaryServerInfo این دیتا تایپ
// اسم کامل متدی است که در حال اجراست
// مثال : /user.UserService/GetUser

// grpc.UnaryHandler این دیتا تایپ
// در اصل خود rpc یا همین تابع زیر است
// rpc GetUser(GetUserRequest) returns (GetUserResponse);

func RecoveryInterceptor() grpc.UnaryServerInterceptor {

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (
		resp any,
		err error,
	) {

		// بعد از هربار اتمام تابع
		defer func() {
			// اگر پنیک داشته باشیم ریکاور میگیرتش و در متغیر قرارش میده
			if r := recover(); r != nil {
				log.Printf(
					"panic recovered in %s: %v\n%s",
					info.FullMethod, // نام کامل متد
					r,               // اطلاعات دلیل پنیک درخواست
					debug.Stack(),   // کلیت مسیر رسیدن به یک پنیک
				)

				err = status.Error(
					codes.Internal,
					"internal server error",
				)
			}
		}()

		return handler(ctx, req) // کل درخواست رو میده بیرون
	}
}
