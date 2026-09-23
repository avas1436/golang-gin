// services/payment-service/internal/service/helper.go

package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"pkg/auth"
	appErrors "pkg/errors"
	"pkg/postgres"

	"payment-service/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// simulateGatewayAndFinalize نتیجه‌ی شبیه‌سازی‌شده‌ی درگاه بانکی را
// روی رکورد پرداخت اعمال کرده و رویداد نتیجه (payment.completed یا
// payment.failed) را منتشر می‌کند.
//
// خطاهای این مرحله فقط لاگ می‌شوند و به بالا برنمی‌گردند: پرداخت
// already در دیتابیس با وضعیت pending ثبت شده؛ اگر همین‌جا شکست
// بخوریم، سفارش هرگز confirm/cancel نمی‌شود که یک باگ Reconciliation
// جداگانه است (خارج از حوصله‌ی همین متد)، نه دلیلی برای از دست دادن
// خودِ رکورد پرداخت.
func (
	s *PaymentService,
) simulateGatewayAndFinalize(
	ctx context.Context,
	payment *model.Payment,
) {

	const gatewayName = "sandbox-gateway"

	// ۸۰٪ نرخ موفقیت برای شبیه‌سازی سناریوهای واقعی
	success := rand.Intn(100) < 80

	err := postgres.WithTx(ctx, s.pool, func(tx pgx.Tx) error {

		if success {
			refID := fmt.Sprintf("ref_%s", uuid.NewString())
			return s.paymentRepo.UpdateStatus(
				ctx,
				tx,
				payment.ID,
				model.PaymentStatusCompleted,
				strPtr(gatewayName),
				strPtr(refID),
				nil,
			)
		}

		reason := "gateway declined the transaction"
		return s.paymentRepo.UpdateStatus(
			ctx,
			tx,
			payment.ID,
			model.PaymentStatusFailed,
			nil,
			nil,
			&reason,
		)
	})

	if err != nil {
		log.Printf(
			"payment-service: failed to finalize payment %s: %v",
			payment.ID,
			err,
		)
		return
	}

	if success {
		if err := s.publisher.PublishPaymentCompleted(
			ctx,
			payment.ID,
			payment.OrderID,
			gatewayName,
			"",
		); err != nil {
			log.Printf(
				"payment-service: failed to publish payment.completed for %s: %v",
				payment.ID,
				err,
			)
		}
		return
	}

	if err := s.publisher.PublishPaymentFailed(
		ctx,
		payment.ID,
		payment.OrderID,
		"gateway declined the transaction",
	); err != nil {
		log.Printf(
			"payment-service: failed to publish payment.failed for %s: %v",
			payment.ID,
			err,
		)
	}
}

func strPtr(s string) *string { return &s }

const roleAdmin = "admin"

// requireAdmin بررسی می‌کند که درخواست‌کننده نقش ادمین داشته باشد؛
// همان الگوی requireAdmin در product-service
func requireAdmin(ctx context.Context) error {

	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return appErrors.New(
			appErrors.KindUnauthenticated,
			"authentication required",
		)
	}

	if claims.Role != roleAdmin {
		return appErrors.New(
			appErrors.KindPermissionDenied,
			"only admins can perform this action",
		)
	}

	return nil
}
