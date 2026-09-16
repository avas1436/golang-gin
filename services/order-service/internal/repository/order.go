// services/order-service/internal/repository/order.go

package repository

import (
	"context"
	stdErrors "errors"
	"time"

	appErrors "pkg/errors"
	"pkg/postgres"

	"order-service/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolationCode = "23505"

type OrderRepository interface {

	// سفارش و آیتم هایش باید در یک تراکنش ایجاد شوند
	Create(ctx context.Context, order *model.Order) error

	// سفارش را به‌همراه آیتم‌هایش برمی‌گرداند
	GetByID(
		ctx context.Context,
		id uuid.UUID,
	) (
		*model.Order,
		error,
	)

	// برای مشاهده سفارش های من در حساب کاربران است
	ListByUser(
		ctx context.Context,
		userID uuid.UUID,
		limit int,
		offset int,
	) (
		[]*model.Order,
		error,
	)

	// برای تغییر یک سفارش در حال آماده سازی به کنسل شده یا پرداخت شده
	// نکته اینجاست که لایه مدل تنها اجازه تغییر از درحال آماده سازی به کنسل
	// یا تایید را میدهد. این کار باعث ایجاد یک قفل در لایه اپ است
	UpdateStatus(
		ctx context.Context,
		id uuid.UUID,
		next model.OrderStatus,
	) error
}

// همه چیز در اینجا در یک تراکنش انجام میشود
type orderRepository struct {
	pool *pgxpool.Pool
}

// ساخت یک رپوزیتوری سفارشات
func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &orderRepository{pool: pool}
}

// ساخت یک سفارش در یک تراکنش
func (
	r *orderRepository,
) Create(
	ctx context.Context,
	order *model.Order,
) error {

	// بررسی خالی نبودن سفارش
	if order == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"order cannot be nil",
		)
	}

	// اعتبار سنجی دیتای سفارش
	if err := order.Validate(); err != nil {
		return err
	}

	return postgres.WithTx(
		ctx,
		r.pool,
		func(tx pgx.Tx) error {

			// درج سفارش اصلی
			insertOrder := `
			INSERT INTO orders (user_id, total_amount)
			VALUES ($1, $2)
			RETURNING id, status, created_at, updated_at
		`

			if err := tx.QueryRow(
				ctx,
				insertOrder,
				order.UserID,
				order.TotalAmount,
			).Scan(
				&order.ID,
				&order.Status,
				&order.CreatedAt,
				&order.UpdatedAt,
			); err != nil {
				return appErrors.Wrap(
					appErrors.KindInternal,
					err,
					"failed to insert order",
				)
			}

			// درج دسته‌جمعی آیتم‌ها برای کاهش تاخیر شبکه
			n := len(order.Items)

			if n > 0 {
				orderIDs := make([]uuid.UUID, n)
				productIDs := make([]uuid.UUID, n)
				productNames := make([]string, n)
				unitPrices := make([]int64, n)
				quantities := make([]int32, n)

				// ساخت آرایه های جدید از محتوی سفارش
				for i, item := range order.Items {
					item.OrderID = order.ID
					orderIDs[i] = item.OrderID
					productIDs[i] = item.ProductID
					productNames[i] = item.ProductName
					unitPrices[i] = item.UnitPrice
					quantities[i] = item.Quantity
				}

				insertItemsQuery := `
				INSERT INTO order_items (
					order_id, 
					product_id, 
					product_name, 
					unit_price, 
					quantity
				)
				SELECT 
					unnest($1::uuid[]), 
					unnest($2::uuid[]), 
					unnest($3::text[]), 
					unnest($4::bigint[]), 
					unnest($5::int[])
				RETURNING id, product_id, created_at
			`

				rows, err := tx.Query(
					ctx,
					insertItemsQuery,
					orderIDs,
					productIDs,
					productNames,
					unitPrices,
					quantities,
				)

				if err != nil {
					var pgErr *pgconn.PgError
					if stdErrors.As(
						err,
						&pgErr,
					) && pgErr.Code == uniqueViolationCode {

						return appErrors.New(
							appErrors.KindInvalidInput,
							"duplicate product in order items",
						)
					}

					return appErrors.Wrap(
						appErrors.KindInternal,
						err,
						"failed to insert order items",
					)
				}
				defer rows.Close()

				// اسکن نتایج و تطبیق با آیتم‌های سفارش بر اساس آیدی محصول
				//
				// چون ترتیب RETURNING تضمین‌شده نیست، از map استفاده می‌کنیم.
				itemIndexByProduct := make(map[uuid.UUID]int, n)
				for i := range order.Items {
					itemIndexByProduct[order.Items[i].ProductID] = i
				}

				scanned := 0
				for rows.Next() {

					var (
						itemID    uuid.UUID
						productID uuid.UUID
						createdAt time.Time
					)

					if err := rows.Scan(&itemID, &productID, &createdAt); err != nil {

						return appErrors.Wrap(
							appErrors.KindInternal,
							err,
							"failed to scan inserted order item",
						)

					}

					idx, ok := itemIndexByProduct[productID]
					if !ok {
						return appErrors.New(
							appErrors.KindInternal,
							"returned product_id not found in original order items",
						)
					}

					order.Items[idx].ID = itemID
					order.Items[idx].CreatedAt = createdAt
					scanned++
				}

				if err := rows.Err(); err != nil {
					return appErrors.Wrap(
						appErrors.KindInternal,
						err,
						"error iterating inserted order items",
					)
				}

				// بررسی تطابق تعداد ردیف‌های درج‌شده با آیتم‌های سفارش
				if scanned != n {
					return appErrors.New(
						appErrors.KindInternal,
						"mismatch between inserted rows and order items count",
					)
				}
			}

			return nil
		},
	)
}

// دریافت اطلاعات یک سفارش بدون جزییات آیتم ها
func (
	r *orderRepository,
) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (
	*model.Order,
	error,
) {

	order := &model.Order{}

	orderQuery := `
		SELECT id, user_id, status, total_amount, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	// دریافت اطلاعات اولیه سفارش
	err := r.pool.QueryRow(
		ctx,
		orderQuery,
		id,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if stdErrors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.New(
				appErrors.KindNotFound,
				"order not found",
			)
		}

		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to get order by id",
		)
	}

	// دریافت جزییات آیتم ها و قرار دادن در لیست
	items, err := r.getItems(ctx, id)
	if err != nil {
		return nil, err
	}

	order.Items = items

	return order, nil
}

// دریافت جزییات آیتم های یک سفارش
func (
	r *orderRepository,
) getItems(
	ctx context.Context,
	orderID uuid.UUID,
) (
	[]*model.OrderItem,
	error,
) {

	query := `
		SELECT 
			id, 
			order_id, 
			product_id, 
			product_name, 
			unit_price, 
			quantity, 
			created_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at
	`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to load order items",
		)
	}
	defer rows.Close()

	items := make([]*model.OrderItem, 0)

	for rows.Next() {
		item := &model.OrderItem{}

		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductName,
			&item.UnitPrice,
			&item.Quantity,
			&item.CreatedAt,
		); err != nil {
			return nil, appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to scan order item row",
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"error iterating order item rows",
		)
	}

	return items, nil
}

// دریافت سفارش های یک کاربر
func (
	r *orderRepository,
) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
	offset int,
) (
	[]*model.Order, error,
) {

	if limit <= 0 {
		limit = 20
	}

	query := `
		SELECT id, user_id, status, total_amount, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to list orders by user",
		)
	}
	defer rows.Close()

	orders := make([]*model.Order, 0)

	for rows.Next() {
		o := &model.Order{}

		if err := rows.Scan(
			&o.ID,
			&o.UserID,
			&o.Status,
			&o.TotalAmount,
			&o.CreatedAt,
			&o.UpdatedAt,
		); err != nil {
			return nil, appErrors.Wrap(
				appErrors.KindInternal,
				err,
				"failed to scan order row",
			)
		}

		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"error iterating order rows",
		)
	}

	return orders, nil
}

// آپدیت وضعیت یک سفارش
func (
	r *orderRepository,
) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	next model.OrderStatus,
) error {

	query := `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3
	`

	result, err := r.pool.Exec(
		ctx,
		query,
		next,
		id,
		model.OrderStatusPending,
	)
	if err != nil {
		return appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to update order status",
		)
	}

	if result.RowsAffected() == 0 {
		// یا سفارش وجود ندارد، یا از قبل در یک وضعیت نهایی است.
		// KindAlreadyExists اینجا به این معناست که «این گذار وضعیت
		// قبلاً اتفاق افتاده یا دیگر ممکن نیست»، نه یک خطای واقعاً
		// غیرمنتظره
		return appErrors.New(
			appErrors.KindAlreadyExists,
			"order is not in a pending state",
		)
	}

	return nil
}
