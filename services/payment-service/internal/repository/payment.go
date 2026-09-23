// services/payment-service/internal/repository/payment.go

package repository

import (
	"context"
	stdErrors "errors"

	appErrors "pkg/errors"
	"pkg/postgres"

	"payment-service/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PaymentRepository رابط کار با دیتابیس برای مدیریت پرداخت‌ها است.
// این سرویس مستقیم توسط کاربر نهایی فراخوانی نمی‌شود، بلکه داده‌ها را
// از طریق پیام‌های RabbitMQ (مثل ایجاد سفارش) یا درخواست‌های gRPC (برای دیباگ) دریافت می‌کند.
type PaymentRepository interface {

	// Create یک رکورد پرداخت جدید را در وضعیت اولیه (pending) ثبت می‌کند.
	// برای حفظ یکپارچگی داده‌ها، این متد باید حتماً داخل یک تراکنش دیتابیس (tx) اجرا شود.
	Create(
		ctx context.Context,
		payment *model.Payment,
	) error

	// GetByOrderID آخرین رکورد پرداخت مربوط به یک سفارش خاص را برمی‌گرداند.
	GetByOrderID(
		ctx context.Context,
		orderID uuid.UUID,
	) (
		*model.Payment,
		error,
	)

	// UpdateStatus وضعیت یک پرداخت را تغییر می‌دهد (مثلاً از pending به success یا failed).
	// این متد نیازمند تراکنش دیتابیس (tx) است.
	UpdateStatus(
		ctx context.Context,
		tx pgx.Tx,
		id uuid.UUID,
		next model.PaymentStatus,
		gatewayName *string,
		gatewayRefID *string,
		failureReason *string,
	) error
}

type paymentRepository struct {
	db postgres.DBTX
}

func NewPaymentRepository(db postgres.DBTX) PaymentRepository {
	return &paymentRepository{db: db}
}

// Create رکورد جدید پرداخت را ثبت می‌کند.
func (
	r *paymentRepository,
) Create(
	ctx context.Context,
	payment *model.Payment,
) error {

	// بررسی می‌کند که آیا داده‌ی پرداخت معتبر است یا خیر
	if payment == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment cannot be nil",
		)
	}

	// اعتبارسنجی داده‌ی پرداخت
	if err := payment.Validate(); err != nil {
		return err
	}

	query := `
		INSERT INTO payments (
			id,
			order_id,
			user_id,
			amount,
			currency,
			status,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`

	// اجرای کوئری برای ثبت پرداخت
	err := r.db.QueryRow(
		ctx,
		query,
		payment.ID,
		payment.OrderID,
		payment.UserID,
		payment.Amount,
		payment.Currency,
		payment.Status,
		payment.Metadata,
	).Scan(&payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {

		// PostgreSQL خطاهای خودش را با *pgconn.PgError
		// برمی‌گرداند.
		var pgErr *pgconn.PgError

		// بررسی خطای تکراری بودن (Unique Violation در PostgreSQL با کد 23505)
		if stdErrors.As(err, &pgErr) &&
			pgErr.Code == "23505" {

			return appErrors.New(
				appErrors.KindAlreadyExists,
				"an active payment already exists for this order",
			)
		}

		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to create payment in database",
		)
	}

	return nil
}

// GetByOrderID جدیدترین پرداخت ثبت‌شده برای یک سفارش را برمی‌گرداند.
//
// اگر یک سفارش چندین بار تلاش برای پرداخت داشته باشد (مثلاً پرداخت‌های ناموفق قبلی)،
// این کوئری با استفاده از ORDER BY created_at DESC LIMIT 1 آخرین تلاش را برمی‌گرداند.
func (
	r *paymentRepository,
) GetByOrderID(
	ctx context.Context,
	orderID uuid.UUID,
) (
	*model.Payment,
	error,
) {

	query := `
		SELECT
			id,
			order_id,
			user_id,
			amount,
			currency,
			status,
			gateway_name,
			gateway_ref_id,
			failure_reason,
			metadata,
			created_at,
			updated_at
		FROM payments
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	p := &model.Payment{}

	err := r.db.QueryRow(ctx, query, orderID).Scan(
		&p.ID,
		&p.OrderID,
		&p.UserID,
		&p.Amount,
		&p.Currency,
		&p.Status,
		&p.GatewayName,
		&p.GatewayRefID,
		&p.FailureReason,
		&p.Metadata,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if stdErrors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.New(
				appErrors.KindNotFound,
				"payment not found for this order",
			)
		}

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to get payment by order id",
		)
	}

	return p, nil
}

// UpdateStatus وضعیت پرداخت را فقط در صورتی به‌روزرسانی می‌کند که وضعیت فعلی آن pending باشد.
//
// شرط `status = pending` مانع از این می‌شود که دو پردازش هم‌زمان (Race Condition)
// بتوانند وضعیت یک پرداخت تعیین‌تکلیف‌شده را دوباره تغییر دهند.
func (
	r *paymentRepository,
) UpdateStatus(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
	next model.PaymentStatus,
	gatewayName *string,
	gatewayRefID *string,
	failureReason *string,
) error {

	query := `
		UPDATE payments
		SET
			status         = $1,
			gateway_name   = $2,
			gateway_ref_id = $3,
			failure_reason = $4,
			updated_at     = NOW()
		WHERE id = $5
		  AND status = $6
	`

	result, err := tx.Exec(
		ctx,
		query,
		next,
		gatewayName,
		gatewayRefID,
		failureReason,
		id,
		model.PaymentStatusPending,
	)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to update payment status",
		)
	}

	if result.RowsAffected() == 0 {
		// یا رکورد وجود ندارد، یا از قبل در یک وضعیت نهایی/غیر از
		// pending است؛ یعنی این گذار وضعیت قبلاً اتفاق افتاده یا
		// دیگر ممکن نیست
		return appErrors.New(
			appErrors.KindAlreadyExists,
			"payment is not in a pending state",
		)
	}

	return nil
}
