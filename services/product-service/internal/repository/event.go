// services/product-service/internal/repository/event.go

package repository

import (
	"context"

	appErrors "pkg/errors"
	"pkg/postgres"

	"github.com/google/uuid"
)

// برای جلوگیری از انجام دوباره یک تایید سفارش این رپوزیتوری وجود دارد
type EventRepository interface {
	MarkProcessed(
		ctx context.Context,
		eventID uuid.UUID,
		eventType string,
		productID uuid.UUID,
	) (
		alreadyProcessed bool, err error,
	)
}

type eventRepository struct {
	db postgres.DBTX
}

func NewEventRepository(db postgres.DBTX) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) MarkProcessed(
	ctx context.Context,
	eventID uuid.UUID,
	eventType string,
	productID uuid.UUID,
) (
	bool, error,
) {

	query := `
		INSERT INTO processed_events (event_id, event_type, product_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (event_id) DO NOTHING
	`

	result, err := r.db.Exec(ctx, query, eventID, eventType, productID)
	if err != nil {
		return false, appErrors.Wrap(
			appErrors.KindInternal,
			err,
			"failed to record processed event",
		)
	}

	alreadyProcessed := result.RowsAffected() == 0

	return alreadyProcessed, nil
}
