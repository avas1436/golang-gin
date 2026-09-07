// api/internal/handler/auth/endpoints.go

package auth

import (
	"net/http"

	"api/internal/client"
	"api/internal/httperrors"
	pb "pkg/proto/user"

	"github.com/gin-gonic/gin"
)

// Register ثبت نام کاربر جدید
// @Summary      ثبت نام کاربر
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body registerRequestBody true "اطلاعات ثبت نام"
// @Success      200 {object} authResponseBody
// @Failure      400 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var body registerRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	resp, err := h.userClient.Register(c.Request.Context(), &pb.RegisterRequest{
		PhoneNumber: body.PhoneNumber,
		Email:       body.Email,
		Password:    body.Password,
		FullName:    body.FullName,
	})
	if err != nil {
		httperrors.Respond(c, err)
		return
	}

	h.respondWithAuth(c, resp)
}

// PasswordLogin ورود با رمز عبور
// @Summary      ورود با رمز عبور
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body passwordLoginRequestBody true "اطلاعات ورود"
// @Success      200 {object} authResponseBody
// @Failure      400 {object} errorResponse
// @Failure      401 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /api/v1/auth/login/password [post]
func (h *Handler) PasswordLogin(c *gin.Context) {
	var body passwordLoginRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	resp, err := h.userClient.PasswordLogin(c.Request.Context(), &pb.PasswordLoginRequest{
		Identifier: body.Identifier,
		Password:   body.Password,
	})
	if err != nil {
		httperrors.Respond(c, err)
		return
	}

	h.respondWithAuth(c, resp)
}

// OTPLogin درخواست کد یکبار مصرف
// @Summary      ارسال کد OTP
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body otpLoginRequestBody true "شماره همراه"
// @Success      200 {object} otpLoginResponseBody
// @Failure      400 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /api/v1/auth/login/otp [post]
func (h *Handler) OTPLogin(c *gin.Context) {
	var body otpLoginRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	resp, err := h.userClient.OTPLogin(c.Request.Context(), &pb.OTPLoginRequest{
		PhoneNumber: body.PhoneNumber,
	})
	if err != nil {
		httperrors.Respond(c, err)
		return
	}

	c.JSON(
		http.StatusOK,
		otpLoginResponseBody{
			ChallengeID:      resp.ChallengeId,
			ExpiresInSeconds: int64(resp.ExpiresInSeconds),
		},
	)
}

// VerifyOTP تایید کد یکبار مصرف
// @Summary      تایید کد OTP
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body verifyOTPRequestBody true "شناسه چالش و کد"
// @Success      200 {object} authResponseBody
// @Failure      400 {object} errorResponse
// @Failure      401 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /api/v1/auth/login/otp/verify [post]
func (h *Handler) VerifyOTP(c *gin.Context) {
	var body verifyOTPRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	resp, err := h.userClient.VerifyOTP(c.Request.Context(), &pb.VerifyOTPRequest{
		OtpChallengeId: body.ChallengeID,
		OtpCode:        body.Code,
	})
	if err != nil {
		httperrors.Respond(c, err)
		return
	}

	h.respondWithAuth(c, resp)
}

// RefreshToken تجدید توکن
// @Summary      تجدید توکن دسترسی
// @Tags         Auth
// @Produce      json
// @Success      200 {object} authResponseBody
// @Failure      401 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /api/v1/auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName)
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "missing refresh token"})
		return
	}

	resp, err := h.userClient.RefreshToken(c.Request.Context(), &pb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		h.clearRefreshCookie(c)
		httperrors.Respond(c, err)
		return
	}

	h.respondWithAuth(c, resp)
}

// Logout خروج از حساب
// @Summary      خروج از حساب کاربری
// @Tags         Auth
// @Produce      json
// @Success      200 {object} messageResponse
// @Failure      401 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /api/v1/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie(refreshCookieName)

	if refreshToken != "" {
		if _, err := h.userClient.Logout(c.Request.Context(), &pb.LogoutRequest{
			RefreshToken: refreshToken,
		}); err != nil {
			h.clearRefreshCookie(c)
			httperrors.Respond(c, err)
			return
		}
	}

	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, messageResponse{Message: "logged out"})
}

// GetUser دریافت اطلاعات پروفایل کاربر
// @Summary      دریافت اطلاعات کاربر
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "شناسه کاربر"
// @Success      200 {object} userBody
// @Failure      401 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /api/v1/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	id := c.Param("id")

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
