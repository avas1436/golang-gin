// api/internal/handler/auth/handler.go

package auth

import (
	"net/http"

	"api/config"

	pb "pkg/proto/user"

	"github.com/gin-gonic/gin"
)

const (
	// کلید نام رفرش توکن در کوکی
	refreshCookieName = "refresh_token"

	// کوکی عمداً فقط روی مسیر /api/v1/auth معتبر است، نه کل سایت —
	// یعنی حتی اگر یک endpoint دیگر آسیب‌پذیر باشد، این کوکی برایش
	// ارسال نمی‌شود.
	refreshCookiePath = "/api/v1/auth"
)

type Handler struct {
	userClient pb.UserServiceClient
	cookieCfg  config.CookieConfig
}

func NewHandler(
	userClient pb.UserServiceClient,
	cookieCfg config.CookieConfig,
) *Handler {

	return &Handler{
		userClient: userClient,
		cookieCfg:  cookieCfg,
	}

}

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshCookieName,
		token,
		h.cookieCfg.RefreshMaxAgeSeconds,
		refreshCookiePath,
		h.cookieCfg.Domain,
		h.cookieCfg.Secure,
		true,
	)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshCookieName,
		"",
		-1,
		refreshCookiePath,
		h.cookieCfg.Domain,
		h.cookieCfg.Secure,
		true,
	)
}

func (
	h *Handler,
) respondWithAuth(
	c *gin.Context,
	auth *pb.AuthResponse,
) {

	h.setRefreshCookie(c, auth.RefreshToken)
	c.JSON(
		http.StatusOK,
		authResponseBody{
			AccessToken: auth.AccessToken,
			ExpiresIn:   auth.ExpiresIn,
			User:        toUserBody(auth.User),
		},
	)

}
