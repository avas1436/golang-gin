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
	"payment-service/internal/repository"

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

	const failureReason = "gateway declined the transaction"

	// refID قبل از تراکنش ساخته می‌شود تا همان مقداری که در دیتابیس
	// ذخیره می‌شود، عیناً در رویداد payment.completed هم منتشر شود.
	var refID string
	if success {
		refID = fmt.Sprintf("ref_%s", uuid.NewString())
	}

	err := postgres.WithTx(
		ctx,
		s.pool,
		func(tx pgx.Tx) error {

			txPaymentRepo := repository.NewPaymentRepository(tx)

			if success {

				// استفاده از متد مدل به جای وارد کردن دستی مقادیر
				if err := payment.MarkCompleted(gatewayName, refID); err != nil {
					return err
				}

			} else {
				// اعمال State Transition روی Domain Model
				if err := payment.MarkFailed(failureReason); err != nil {
					return err
				}
			}

			// بروزرسانی دیتابیس با مدل تغییر یافته
			return txPaymentRepo.UpdateFromPending(ctx, payment)

		},
	)

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
			refID,
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
		failureReason,
	); err != nil {
		log.Printf(
			"payment-service: failed to publish payment.failed for %s: %v",
			payment.ID,
			err,
		)
	}
}

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
