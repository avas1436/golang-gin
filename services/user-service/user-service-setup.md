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

## 6. ساخت مایگریشن های دیتا بیس

- ✅ ابتدا نصب golang-migrate برای مدیریت مایگریشن ها

  ```
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```

- ✅ ایجاد اولین مایگریشن ها برای جدول کاربر و رفرش توکن

  ```
  cd services/user-service
  migrate create -ext sql -dir migrations -seq create_users_table
  migrate create -ext sql -dir migrations -seq create_refresh_tokens_table
  ```

- ✅ حالا دقیقا مطابق جدولی که ایجاد کردیم باید محتوی فایل های مایگریشن رو پرکنیم و مقادیر موجود در جداول و ویژگی هر ستون و ایندکس ها و .. رو به صورت دستی بر خلاف کد های پایتونی باید دستی بنویسیم.

- ✅ حالا برای اجرای مایگریشن ها نیاز داریم به دیتا بیس وصل شویم پس برای اجرای آزمایشی به صورت local ابتدا وارد دیتایس میشویم و یک کاربر مخصوص این سرویس ایجاد میکنیم.

<div dir="ltr">

```
  -- در ترمینال ابونتو
  sudo -u postgres psql

  -- ساخت یوزر جدید
  CREATE ROLE user_service WITH LOGIN PASSWORD 'strong_password';

  -- ساخت دیتا بیس جدید
  CREATE DATABASE user_service_db OWNER user_service;
```

</div>

- ✅ در مرحله آخر هم برای اجرا و اعمال مایگریشن ها باید دستور زیر رو مطابق ویژگی های دیتابیس خود بزنیم:

<div dir="ltr">

```
  migrate //
  -database //
  "postgres://user_service:my_password@localhost:5432/user_service_db?sslmode=disable" //
  -path migrations up
```

</div>

## مراحل بعدی (هنوز انجام نشده)

- ⏳ پیاده‌سازی لایه `repository`
- ⏳ پیاده‌سازی لایه `service` (منطق کسب‌وکار + تولید/بررسی JWT و OTP)
- ⏳ پیاده‌سازی لایه `handler` (پیاده‌سازی interface سرور gRPC تولیدشده از proto)
- ⏳ نوشتن `cmd/main.go` برای بالا آوردن سرور gRPC
- ⏳ تنظیم `config` برای خواندن متغیرهای محیطی (پورت، آدرس دیتابیس، آدرس Redis)

</div>
