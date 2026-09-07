// api/internal/client/forward.go

package client

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// این تابع اطلاعات احراز هویتی را از هدر گیت وی به متادیتای مورد
// نیاز سرویس منتقل میکند
// HTTP Header -> gRPC Metadata
func ForwardAuth(
	ctx context.Context,
	authHeader string,
) context.Context {

	if authHeader == "" {
		return ctx
	}

	return metadata.AppendToOutgoingContext(
		ctx,
		"authorization",
		authHeader,
	)
}
