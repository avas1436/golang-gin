// pkg/grpcerrors/error_mapper.go

package grpcerrors

import (
	appErrors "pkg/errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
)

// FromAppError خطاهای داخلی برنامه که با pkg/errors ساخته شده‌اند
// را به یک خطای استاندارد gRPC تبدیل می‌کند.
func FromAppError(err error, domain string) error {

	// اگر اصلاً خطایی وجود نداشته باشد，
	// نباید یک gRPC error جدید بسازیم.
	if err == nil {
		return nil
	}

	// از خطای داخلی برنامه نوع خطا را استخراج می‌کنیم.
	kind := appErrors.GetKind(err)

	switch kind {

	// KindInvalidInput == > codes.InvalidArgument
	case appErrors.KindInvalidInput:
		return withDetails(
			codes.InvalidArgument,
			err,
			&errdetails.BadRequest{
				FieldViolations: []*errdetails.BadRequest_FieldViolation{
					{
						Field:       "",
						Description: err.Error(),
					},
				},
			},
		)

	// KindUnauthenticated == > codes.Unauthenticated
	case appErrors.KindUnauthenticated:
		return withDetails(
			codes.Unauthenticated,
			err,
			errorInfo(
				"UNAUTHENTICATED",
				domain,
			),
		)

	// KindNotFound == > codes.NotFound
	case appErrors.KindNotFound:

		return withDetails(
			codes.NotFound,
			err,
			errorInfo(
				"NOT_FOUND",
				domain,
			),
		)

	// KindAlreadyExists == > codes.AlreadyExists
	case appErrors.KindAlreadyExists:

		return withDetails(
			codes.AlreadyExists,
			err,
			errorInfo(
				"ALREADY_EXISTS",
				domain,
			),
		)

	// KindPermissionDenied == > codes.PermissionDenied
	case appErrors.KindPermissionDenied:

		return withDetails(
			codes.PermissionDenied,
			err,
			errorInfo(
				"PERMISSION_DENIED",
				domain,
			),
		)

	// بهتره کل این ها به internal تبدیل بشن
	// KindUnknown == > codes.Unknown
	// case appErrors.KindUnknown:

	// 	return withDetails(
	// 		codes.Unknown,
	// 		err,
	// 		errorInfo(
	// 			"UNKNOWN_ERROR",
	// 			domain,
	// 		),
	// 	)

	default:
		// هر خطای ناشناخته ای اینجا تبدیل به internal میشه
		// جزییات خطا هم به دلایل امنیتی به کلاینت داده نمیشه
		st := status.New(
			codes.Internal,
			"internal server error",
		)

		// جزییات ارور هم به آن افزوده میشود تا همان فرمت استاندارد دیگر ارور ها باشد
		stWithDetails, detailErr := st.WithDetails(
			errorInfo(
				"INTERNAL_ERROR",
				domain,
			),
		)

		// اگر شد با جزییات بیرون داده میشه
		if detailErr == nil {
			return stWithDetails.Err()
		}

		// اگه نشد همون فرمت خام
		return st.Err()
	}
}

// errorInfo یک ErrorInfo استاندارد gRPC می‌سازد.
// هدف از این تابع اضافه کردن جزییات به خطاست
func errorInfo(reason string, domain string) *errdetails.ErrorInfo {

	return &errdetails.ErrorInfo{
		// Reason نوع یا علت خطا را مشخص می‌کند.
		Reason: reason,

		// Domain مشخص می‌کند این خطا مربوط به کدام سرویس یا دامنه است.
		Domain: domain,
	}
}

// withDetails یک gRPC Status ایجاد می‌کند
// اول یک ارور خام میسازد و بعد خودکار بهش جزییات اضافه میکند
func withDetails(
	code codes.Code,
	err error,
	detail protoadapt.MessageV1,
) error {

	// ساخت اولیه ارور
	st := status.New(
		code,
		err.Error(),
	)

	// افزودن جزییات
	stWithDetails, detailErr := st.WithDetails(detail)

	// اگر شد با جزییات بیرون داده میشه
	if detailErr == nil {
		return stWithDetails.Err()
	}

	// اگه نشد همون فرمت خام
	return st.Err()
}
