// services/order-service/internal/repository/order.go

package repository

import (
	"context"
	stdErrors "errors"

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

	if order == nil {
		return appErrors.New(
			appErrors.KindInvalidInput,
			"order cannot be nil",
		)
	}

	if err := order.Validate(); err != nil {
		return err
	}

	err := postgres.WithTx(ctx, r.pool, func(tx pgx.Tx) error {

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

		insertItem := `
			INSERT INTO order_items (
				order_id, 
				product_id, 
				product_name, 
				unit_price, 
				quantity
			)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at
		`

		for _, item := range order.Items {

			item.OrderID = order.ID

			if err := tx.QueryRow(
				ctx,
				insertItem,
				item.OrderID,
				item.ProductID,
				item.ProductName,
				item.UnitPrice,
				item.Quantity,
			).Scan(
				&item.ID,
				&item.CreatedAt,
			); err != nil {

				// uq_order_product جلوی دو ردیف برای یک محصول در
				// یک سفارش را می‌گیرد؛ این را به یک خطای دامنه‌ی
				// خوانا تبدیل می‌کنیم، نه یک pgconn.PgError خام
				var pgErr *pgconn.PgError
				if stdErrors.As(err, &pgErr) &&
					pgErr.Code == uniqueViolationCode {

					return appErrors.New(
						appErrors.KindInvalidInput,
						"duplicate product in order items",
					)
				}

				return appErrors.Wrap(
					appErrors.KindInternal,
					err,
					"failed to insert order item",
				)
			}
		}

		return nil
	})

	return err
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
		SELECT id, order_id, product_id, product_name,
		       unit_price, quantity, created_at
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
		ctx, query, next, id, model.OrderStatusPending,
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
