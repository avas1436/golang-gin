// services/payment-service/internal/repository/worker.go

package repository

import (
	"context"
	"payment-service/internal/model"
	appErrors "pkg/errors"
	"time"

	"github.com/google/uuid"
)

// شناسایی پرداخت‌هایی که کاربر وارد درگاه شده اما عملیات را تکمیل نکرده
// و کالبک بانک فراخوانی نشده است
// با شرط status = 'awaiting'
func (
	r *paymentRepository,
) GetStaleAwaitingPayments(
	ctx context.Context,
	timeout time.Duration,
	limit int,
) (
	[]*model.Payment,
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
			authority,
			redirect_url,
			failure_reason,
			metadata,
			created_at,
			updated_at
		FROM payments
		WHERE status = $1
		  AND updated_at < $2
		ORDER BY updated_at ASC
		LIMIT $3
	`

	cutoffTime := time.Now().Add(-timeout)

	rows, err := r.db.Query(
		ctx,
		query,
		model.PaymentStatusAwaitingPayment,
		cutoffTime,
		limit,
	)

	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to query stale awaiting payments",
		)
	}
	defer rows.Close()

	var payments []*model.Payment
	for rows.Next() {
		p := &model.Payment{}
		err := rows.Scan(
			&p.ID,
			&p.OrderID,
			&p.UserID,
			&p.Amount,
			&p.Currency,
			&p.Status,
			&p.GatewayName,
			&p.GatewayRefID,
			&p.Authority,
			&p.RedirectURL,
			&p.FailureReason,
			&p.Metadata,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to scan stale payment",
			)
		}
		payments = append(payments, p)
	}

	if err := rows.Err(); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"error iterating stale payments rows",
		)
	}

	return payments, nil
}

// MarkExpiredAtomic پرداخت را در وضعیت awaiting به expired تبدیل
// می‌کند و رویداد را در همان تراکنش در outbox ثبت می‌کند.
//
// Atomicity: اگر ثبت رویداد در outbox شکست بخورد، تغییر وضعیت
// پرداخت هم rollback می‌شود؛ و برعکس. این تضمین می‌کند که هیچ‌وقت
// پرداخت expired بدون رویداد متناظرش در outbox نداشته باشیم
func (
	r *paymentRepository,
) MarkExpiredAtomic(
	ctx context.Context,
	paymentID uuid.UUID,
	outboxEvent *model.OutboxEvent,
) error {

	if paymentID == uuid.Nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"payment id cannot be empty",
		)
	}

	if outboxEvent == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"outbox event cannot be nil",
		)
	}

	// استفاده از WithTx نیازمند دسترسی به *pgxpool.Pool است، اما
	// این repository روی DBTX کار می‌کند. پس تراکنش را از بیرون
	// (در لایه service) باز می‌کنیم و این متد را با یک tx
	// صدا می‌زنیم. این متد فقط دو عملیات را روی همان DBTX انجام
	// می‌دهد و اگر DBTX خودش tx باشد، هر دو در همان tx هستند.
	//
	// بنابراین: فرض این است که caller این متد را داخل یک تراکنش
	// صدا می‌زند (مثلاً از طریق postgres.WithTx در service).

	// ۱. تغییر وضعیت پرداخت به expired، فقط اگر در وضعیت awaiting باشد
	updateQuery := `
		UPDATE payments
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3
	`

	result, err := r.db.Exec(
		ctx,
		updateQuery,
		model.PaymentStatusExpired, // 'expired'
		paymentID,
		model.PaymentStatusAwaitingPayment, // 'awaiting'
	)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to mark payment as expired",
		)
	}

	if result.RowsAffected() == 0 {
		return appErrors.New(
			appErrors.KindAlreadyExists,
			"payment is not in awaiting state or does not exist",
		)
	}

	// ۲. ثبت رویداد در outbox در همان تراکنش
	outboxRepo := NewOutboxRepository(r.db)

	if err := outboxRepo.Create(ctx, outboxEvent); err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to write outbox event for expired payment",
		)
	}

	return nil
}
