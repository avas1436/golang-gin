// api/internal/middleware/ratelimit.go

package middleware

import (
	"net/http"

	"pkg/ratelimit"

	"github.com/gin-gonic/gin"
)

// همان محدود کننده مورد استفاده در gRPC اینجا برای Gin اعمال میشود
func RateLimit(
	limiter ratelimit.Limiter,
	limit ratelimit.Limit,
	keyPrefix string,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		key := keyPrefix + ":" + c.ClientIP()

		allowed, err := limiter.Check(c.Request.Context(), key, limit)
		if err != nil {
			// fail-open: مشکل زیرساختی Redis نباید کل Gateway را
			// از کار بیندازد.
			c.Next()
			return
		}

		if !allowed.Allowed {
			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				gin.H{
					"error": "too many requests, please try again later",
				},
			)
			return
		}

		c.Next()
	}
}
