// api/internal/handler/auth_handler.go

package handler

import (
	"net/http"

	"api/config"
	"api/internal/client"
	"api/internal/httperrors"

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

// AuthHandler درخواست‌های REST مربوط به احراز هویت را می‌گیرد و به
// فراخوانی gRPC مناسب روی user-service تبدیل می‌کند.
type AuthHandler struct {
	userClient *client.UserClient
	cookieCfg  config.CookieConfig
}

func NewAuthHandler(
	userClient *client.UserClient,
	cookieCfg config.CookieConfig,
) *AuthHandler {

	return &AuthHandler{
		userClient: userClient,
		cookieCfg:  cookieCfg,
	}
}

// این تابع رفرش توکن را در کوکی قرار میدهد
func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		refreshCookieName,
		token,
		h.cookieCfg.RefreshMaxAgeSeconds,
		refreshCookiePath,
		h.cookieCfg.Domain,
		h.cookieCfg.Secure,
		true, // httpOnly
	)
}

// این تابع کوکی را پاک میکند
func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {

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

// فرمت کلی پاسخ در اند پوینت های احراز هویت
type authResponseBody struct {
	AccessToken string    `json:"access_token"`
	ExpiresIn   int64     `json:"expires_in"`
	User        *userBody `json:"user,omitempty"`
}

// فرمت نمایش اطلاعات کاربر
type userBody struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	FullName    string `json:"full_name"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// سازنده ساختار خروجی کاربر
func toUserBody(u *pb.User) *userBody {
	if u == nil {
		return nil
	}
	return &userBody{
		ID:          u.Id,
		Email:       u.Email,
		PhoneNumber: u.PhoneNumber,
		FullName:    u.FullName,
		Role:        u.Role.String(),
		CreatedAt:   u.CreatedAt.String(),
		UpdatedAt:   u.UpdatedAt.String(),
	}
}

// این تابع پاسخ احراز هویت را میسازد ولی هیچ خروجی ندارد و تنها
// مقادیر را در کوکی و کانتکست قرار میدهد
func (
	h *AuthHandler,
) respondWithAuth(
	c *gin.Context,
	auth *pb.AuthResponse,
) {

	// قرار دادن رفرش توکن در کوکی
	h.setRefreshCookie(c, auth.RefreshToken)

	// قرار دادن مقادیر در کانتکست
	c.JSON(
		http.StatusOK,
		authResponseBody{
			AccessToken: auth.AccessToken,
			ExpiresIn:   auth.ExpiresIn,
			User:        toUserBody(auth.User),
		},
	)
}

// ساختار ورودی درخواست ثبت نام
type registerRequestBody struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Email       string `json:"email"`
	Password    string `json:"password" binding:"required"`
	FullName    string `json:"full_name"`
}

// هدنلر ثبت نام
func (h *AuthHandler) Register(c *gin.Context) {

	// اعتبار سنجی اسکمای ورودی
	var body registerRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "invalid request body",
			},
		)
		return
	}

	resp, err := h.userClient.Register(
		c.Request.Context(),
		&pb.RegisterRequest{
			PhoneNumber: body.PhoneNumber,
			Email:       body.Email,
			Password:    body.Password,
			FullName:    body.FullName,
		},
	)

	if err != nil {
		httperrors.Respond(c, err)
		return
	}

	h.respondWithAuth(c, resp)
}

// ساختار ورودی درخواست ورود با رمز عبور
type passwordLoginRequestBody struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

// هندلر ورود با رمز عبور
func (h *AuthHandler) PasswordLogin(c *gin.Context) {

	// اعتبار سنجی ورودی
	var body passwordLoginRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "invalid request body"},
		)
		return
	}

	resp, err := h.userClient.PasswordLogin(
		c.Request.Context(),
		&pb.PasswordLoginRequest{
			Identifier: body.Identifier,
			Password:   body.Password,
		},
	)
	if err != nil {
		httperrors.Respond(c, err)
		return
	}

	h.respondWithAuth(c, resp)
}

// ورود با otp
type otpLoginRequestBody struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

// هندلر ورورد با otp
func (h *AuthHandler) OTPLogin(c *gin.Context) {

	// اعتبار سنجی مقادیر ورودی
	var body otpLoginRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "invalid request body"},
		)
		return
	}

	resp, err := h.userClient.OTPLogin(
		c.Request.Context(),
		&pb.OTPLoginRequest{
			PhoneNumber: body.PhoneNumber,
		},
	)
	if err != nil {
		httperrors.Respond(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"challenge_id":       resp.ChallengeId,
			"expires_in_seconds": resp.ExpiresInSeconds,
		},
	)
}

// ساختار وروردی تایید کد ارسال شده
type verifyOTPRequestBody struct {
	ChallengeID string `json:"otp_challenge_id" binding:"required"`
	Code        string `json:"otp_code" binding:"required"`
}

// هندلر تایید کد
func (h *AuthHandler) VerifyOTP(c *gin.Context) {

	var body verifyOTPRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "invalid request body"},
		)
		return
	}

	resp, err := h.userClient.VerifyOTP(
		c.Request.Context(),
		&pb.VerifyOTPRequest{
			OtpChallengeId: body.ChallengeID,
			OtpCode:        body.Code,
		},
	)
	if err != nil {
		httperrors.Respond(c, err)
		return
	}

	h.respondWithAuth(c, resp)
}

// این هندلر هیچ نیازی به اسکمای ورودی و خروجی ندارد همه چیز در
// پس زمینه مدیریت میشود
func (h *AuthHandler) RefreshToken(c *gin.Context) {

	refreshToken, err := c.Cookie(refreshCookieName)
	if err != nil || refreshToken == "" {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "missing refresh token"},
		)
		return
	}

	resp, err := h.userClient.RefreshToken(
		c.Request.Context(),
		&pb.RefreshTokenRequest{
			RefreshToken: refreshToken,
		},
	)
	if err != nil {
		// اگر رفرش توکن دیگر معتبر نیست، کوکی کهنه را هم پاک می‌کنیم
		// تا کلاینت مجبور به لاگین دوباره شود، نه تلاش‌های بی‌نتیجه‌ی
		// بیشتر با همان کوکی.
		h.clearRefreshCookie(c)
		httperrors.Respond(c, err)
		return
	}

	// کوکی آپدیت میشود
	h.respondWithAuth(c, resp)
}

// هندلر خروج از حساب
func (h *AuthHandler) Logout(c *gin.Context) {

	refreshToken, _ := c.Cookie(refreshCookieName)

	if refreshToken != "" {
		if _, err := h.userClient.Logout(
			c.Request.Context(),
			&pb.LogoutRequest{
				RefreshToken: refreshToken,
			},
		); err != nil {
			h.clearRefreshCookie(c)
			httperrors.Respond(c, err)
			return
		}
	}

	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// هندلر دریافت اطلاعات کاربر
func (h *AuthHandler) GetUser(c *gin.Context) {

	id := c.Param("id")

	// هدر درخواست مستقیما وارد متادیتا میشود
	ctx := client.ForwardAuth(
		c.Request.Context(),
		c.GetHeader("Authorization"),
	)

	resp, err := h.userClient.GetUser(ctx, &pb.GetUserRequest{Id: id})
	if err != nil {
		httperrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserBody(resp))
}
