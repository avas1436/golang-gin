<!-- docs/user-service-setup.md -->

<div dir="rtl">

# راه‌اندازی User Service

این فایل چک‌لیست مراحلی است که تا این لحظه برای پیاده‌سازی User Service (ارتباط از طریق gRPC با API Gateway و سایر سرویس‌ها) انجام شده است.

## ۱. آماده‌سازی ابزارها

- ✅ بررسی نصب بودن `go` و `protoc`
- ✅ نصب پلاگین‌های تولید کد گو:

  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```

## ۲. ماژول‌بندی Go (طبق ADR شماره ۰۰۰۱)

- ✅ ساخت `go.mod` مستقل برای `pkg`:
  ```bash
  cd pkg && go mod init pkg
  ```
- ✅ ساخت `go.mod` مستقل برای `user-service`:
  ```bash
  cd services/user-service && go mod init user-service
  ```
- ✅ معرفی هر دو ماژول به `go.work` در ریشه پروژه:
  ```bash
  go work use ./pkg
  go work use ./services/user-service
  ```

## ۳. تعریف قرارداد gRPC (Proto)

- ✅ ساخت فایل قرارداد در مسیر:
  ```
  pkg/proto/user/user.proto
  ```
- ✅ تعریف پیام‌ها (messages): `User`, `RegisterRequest/Response`, `LoginRequest/Response`, `VerifyOTPRequest/Response`, `RefreshTokenRequest/Response`, `GetUserRequest`, `LogoutRequest/Response`
- ✅ تعریف enum نقش‌ها برای RBAC: `Role` (`ROLE_ADMIN`, `ROLE_MEMBER`, `ROLE_VIEWER`)
- ✅ استفاده از `google.protobuf.Timestamp` برای فیلد `created_at`
- ✅ تعریف سرویس gRPC `UserService` با متدهای:
  - `Register`
  - `Login`
  - `VerifyOTP`
  - `RefreshToken`
  - `GetUser`
  - `Logout`

## ۴. تولید کد Go از Proto

- ✅ اجرای دستور تولید کد از ریشه پروژه:
  ```bash
  protoc \
    --proto_path=pkg/proto/user \
    --go_out=pkg/proto/user --go_opt=paths=source_relative \
    --go-grpc_out=pkg/proto/user --go-grpc_opt=paths=source_relative \
    pkg/proto/user/user.proto
  ```
- ✅ بررسی ساخته‌شدن فایل‌های خروجی:
  ```
  pkg/proto/user/user.pb.go
  pkg/proto/user/user_grpc.pb.go
  ```
- ✅ بروزرسانی وابستگی‌های ماژول `pkg`:
  ```bash
  cd pkg && go mod tidy
  ```

## ۵. مدل داخلی دامنه (Domain Model)

- ✅ ساخت مدل داخلی جدا از struct های تولیدشده توسط proto، در مسیر:
  ```
  services/user-service/internal/model/user.go
  ```
- ✅ تعریف struct `User` داخلی (شامل `PasswordHash`، که هرگز در proto وجود ندارد)
- ✅ تعریف struct `RefreshToken` داخلی برای نگهداری و ابطال (revoke) توکن‌ها در دیتابیس

## مراحل بعدی (هنوز انجام نشده)

- ⏳ تعریف Migration دیتابیس (جداول `users` و `refresh_tokens`)
- ⏳ پیاده‌سازی لایه `repository`
- ⏳ پیاده‌سازی لایه `service` (منطق کسب‌وکار + تولید/بررسی JWT و OTP)
- ⏳ پیاده‌سازی لایه `handler` (پیاده‌سازی interface سرور gRPC تولیدشده از proto)
- ⏳ نوشتن `cmd/main.go` برای بالا آوردن سرور gRPC
- ⏳ تنظیم `config` برای خواندن متغیرهای محیطی (پورت، آدرس دیتابیس، آدرس Redis)

</div>
