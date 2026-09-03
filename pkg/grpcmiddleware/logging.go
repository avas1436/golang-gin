// pkg/grpcmiddleware/logging.go
package grpcmiddleware

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// فرمت یک میدل ور در gRPC
func LoggingInterceptor() grpc.UnaryServerInterceptor {

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (
		any,
		error,
	) {

		// در این قسمت زمان اجرای یک درخواست محاسبه میشود
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			// [gRPC] /user.UserService/GetUser failed in 120ms: user not found
			log.Printf(
				"[gRPC] %s failed in %s: %v",
				info.FullMethod,
				duration,
				err,
			)
		} else {
			// [gRPC] /user.UserService/GetUser ok in 45ms
			log.Printf(
				"[gRPC] %s ok in %s",
				info.FullMethod,
				duration,
			)
		}

		// جواب کل روند را به gRPC برمیگرداند
		return resp, err
	}
}
