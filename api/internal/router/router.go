// api/internal/router/router.go

package router

import (
	"api/internal/handler"
	"api/internal/middleware"
	"pkg/ratelimit"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handlers شامل تمامی هندلرهای سرویس‌های مختلف است
type Handlers struct {
	Auth *handler.AuthHandler
	// Order        *handler.OrderHandler
	// Product      *handler.ProductHandler
	// Payment      *handler.PaymentHandler
	// Notification *handler.NotificationHandler
}

type Config struct {
	Engine   *gin.Engine
	Handlers Handlers
	Limiter  ratelimit.Limiter
}

func Setup(cfg Config) {
	// ۱. میدل‌ورهای پایه Gin
	cfg.Engine.Use(gin.Recovery(), gin.Logger())

	// ۲. آدرس Swagger UI
	cfg.Engine.GET(
		"/swagger/*any",
		ginSwagger.WrapHandler(swaggerFiles.Handler),
	)

	// ۳. مسیر بررسی سلامت سرور
	cfg.Engine.GET(
		"/health",
		func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		},
	)

	// ۴. گروه‌بندی اصلی API v1
	v1 := cfg.Engine.Group("/api/v1")
	{
		registerAuthRoutes(v1, cfg.Handlers.Auth, cfg.Limiter)
		// registerOrderRoutes(v1, cfg.Handlers.Order, cfg.Limiter)
		// registerProductRoutes(v1, cfg.Handlers.Product, cfg.Limiter)
		// registerPaymentRoutes(v1, cfg.Handlers.Payment, cfg.Limiter)
	}
}

// ثبت ماژولار روت‌های احراز هویت و کاربران
func registerAuthRoutes(
	rg *gin.RouterGroup,
	h *handler.AuthHandler,
	limiter ratelimit.Limiter,
) {

	auth := rg.Group("/auth")
	auth.Use(middleware.RateLimit(
		limiter,
		ratelimit.PerMinute(10),
		"auth",
	),
	)
	{
		auth.POST("/register", h.Register)
		auth.POST("/login/password", h.PasswordLogin)
		auth.POST("/login/otp", h.OTPLogin)
		auth.POST("/login/otp/verify", h.VerifyOTP)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
	}

	users := rg.Group("/users")
	{
		users.GET("/:id", h.GetUser)
	}
}
