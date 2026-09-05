// api/internal/httperrors/httperrors.go

package httperrors

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ساختار خروجی خطای گیت وی
type Body struct {
	// بعد از تبدیل شدن به json نام این قسمت error میشه
	Error string `json:"error"`

	// omitempty یعنی اگر مقدار نداشت کلا در json نباشد
	Reason string `json:"reason,omitempty"`

	// جزییات داخلی تر ارور اینجا قرار میگرد
	Fields []FieldViolation `json:"fields,omitempty"`
}

// مثلا در پایین ترین لایه رپوزیتوری یک ارور داریم و اینجا قرار میگیرد
type FieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
}

// Respond خطای برگشته از یک فراخوانی gRPC (به user-service) را به یک
// پاسخ HTTP مناسب تبدیل می‌کند و مستقیم روی گین می‌نویسد.
//
// این دقیقاً برعکس کاری است که pkg/grpcerrors.FromAppError در سمت
// user-service انجام می‌دهد: آنجا Kind را به codes.Code تبدیل
// کردیم، اینجا codes.Code را به کد وضعیت HTTP تبدیل می‌کنیم. جزئیات
// errdetails (ErrorInfo, BadRequest) که آنجا به خطا ضمیمه شده بودند،
// همینجا دوباره خوانده و به JSON اضافه می‌شوند.
func Respond(c *gin.Context, err error) {

	// اگر ارور از gRPC آمده باشد ok مقدار true دارد
	st, ok := status.FromError(err)

	// اگر منبع خطا gRPC نبود این ارور را بیرون میدهیم
	if !ok {
		c.JSON(
			http.StatusInternalServerError,
			Body{Error: "internal server error"},
		)
		return
	}

	// بدنه خطا متن توضیح خطا میشود
	body := Body{
		Error: st.Message(),
	}

	// اینجا روی پارامتر های خطا حرکت میکنیم
	for _, d := range st.Details() {

		// نوع پارامتر بررسی میشه
		switch detail := d.(type) {

		// دلیل اصلی خطا
		case *errdetails.ErrorInfo:
			body.Reason = detail.Reason

		// جزییات داخلی خطا در فیلد قرار میگیرد
		case *errdetails.BadRequest:
			for _, fv := range detail.FieldViolations {
				body.Fields = append(
					body.Fields,
					FieldViolation{
						Field:       fv.Field,
						Description: fv.Description,
					},
				)
			}
		}
	}

	// در این قسمت فرمت json خروجی ساخته میشود
	c.JSON(httpStatusFromCode(st.Code()), body)
}

// این تابع ارور های gRPC رو به ارور HTTP تبدیل میکند
func httpStatusFromCode(code codes.Code) int {

	switch code {

	case codes.NotFound:
		return http.StatusNotFound

	case codes.AlreadyExists:
		return http.StatusConflict

	case codes.InvalidArgument:
		return http.StatusBadRequest

	case codes.Unauthenticated:
		return http.StatusUnauthorized

	case codes.PermissionDenied:
		return http.StatusForbidden

	case codes.ResourceExhausted:
		return http.StatusTooManyRequests

	default:
		return http.StatusInternalServerError
	}
}
