// pkg/grpcerrors/client.go

package grpcerrors

import (
	appErrors "pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToAppError عکس FromAppError است: وقتی این سرویس خودش یک gRPC
// Client است، خطای برگشتی از سرویس دیگر را به همان appErrors.Kind
// آشنا برمی‌گرداند؛ به این ترتیب لایه‌ی service هیچ‌وقت مجبور نیست
// با google.golang.org/grpc/codes کار کند، فقط با appErrors —
// دقیقاً همان چیزی که برای خطاهای خودِ دیتابیس هم استفاده می‌شود
func ToAppError(err error) error {

	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"non-grpc error from remote service",
		)
	}

	switch st.Code() {

	case codes.InvalidArgument:
		return appErrors.New(appErrors.KindInvalidInput, st.Message())

	case codes.NotFound:
		return appErrors.New(appErrors.KindNotFound, st.Message())

	case codes.AlreadyExists:
		return appErrors.New(appErrors.KindAlreadyExists, st.Message())

	case codes.Unauthenticated:
		return appErrors.New(appErrors.KindUnauthenticated, st.Message())

	case codes.PermissionDenied:
		return appErrors.New(appErrors.KindPermissionDenied, st.Message())

	default:
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"remote service call failed",
		)
	}
}
