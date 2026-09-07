// pkg/errors/errors_test.go

package errors_test

import (
	"testing"

	stdErrors "errors"
	appErrors "pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// تست ساخت یک ارور جدید
func TestNew(t *testing.T) {
	// ساخت یک ارور برای مثال
	err := appErrors.New(appErrors.KindNotFound, "user not found")

	// این دستور بررسی میکند خروجی ارور باشد
	require.Error(t, err, err.Error())

	// این قسمت متن خروجی را میسنجد
	assert.Equal(
		t,
		"user not found",
		err.Error(),
	)

	// این قسمت نوع ارور را اعتبار سنجی میکند
	assert.Equal(
		t,
		appErrors.KindNotFound,
		appErrors.GetKind(err),
	)
}

// بررسی تابع wrap کردن خطا ها
func TestWrap(t *testing.T) {

	// یک ارور پایه برای مثال داریم
	cause := stdErrors.New("connection refused")

	// الان مثلا در لایه دیگر میخواهیم نگهش داریم
	err := appErrors.Wrap(
		appErrors.KindInternal,
		cause,
		"failed to query database",
	)

	// اول بررسی میکنیم اصن ارور هست یا نه
	require.Error(t, err)

	// حالا نوع ارور رو چک میکنیم
	assert.Equal(t, appErrors.KindInternal, appErrors.GetKind(err))

	// چک کردن این در ارور خروجی متن اولیه هست یا نه
	assert.Contains(t, err.Error(), "connection refused")
	assert.Contains(t, err.Error(), "failed to query database")

	// بررسی اینکه به زیری ترین لایه ارور میرسد
	assert.ErrorIs(t, err, cause)
}

// تست دریافت نوع ارور
func TestGetKind(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want appErrors.Kind
	}{
		{
			name: "appErrors.Error with KindNotFound",
			err:  appErrors.New(appErrors.KindNotFound, "not found"),
			want: appErrors.KindNotFound,
		},
		{
			name: "wrapped appErrors.Error keeps its Kind",
			err:  appErrors.Wrap(appErrors.KindAlreadyExists, stdErrors.New("dup"), "exists"),
			want: appErrors.KindAlreadyExists,
		},
		{
			name: "raw standard error is Unknown",
			err:  stdErrors.New("some raw error"),
			want: appErrors.KindUnknown,
		},
		{
			name: "nil error is Unknown",
			err:  nil,
			want: appErrors.KindUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, appErrors.GetKind(tt.err))
		})
	}
}
