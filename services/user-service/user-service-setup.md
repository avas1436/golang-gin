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

<div dir="ltr">

```bash
protoc \
  --proto_path=pkg/proto/user \
  --go_out=pkg/proto/user --go_opt=paths=source_relative \
  --go-grpc_out=pkg/proto/user --go-grpc_opt=paths=source_relative \
  pkg/proto/user/user.proto
```

</div>

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

**چرا مدل داخلی جدا از proto؟** چون proto قراردادیه که بین سرویس‌ها رد و بدل می‌شه — نباید هیچ‌وقت فیلدهای حساس مثل `PasswordHash` یا `TokenHash` توش باشه. مدل داخلی فقط داخل خود سرویس زندگی می‌کنه؛ تبدیل بین این دو (`mapper.go`) تنها جاییه که این دو دنیا به هم وصل می‌شن.

## 6. ساخت مایگریشن های دیتابیس

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

- ✅ حالا دقیقا مطابق جدولی که ایجاد کردیم باید محتوی فایل های مایگریشن رو پرکنیم و مقادیر موجود در جداول و ویژگی هر ستون و ایندکس ها و .. رو به صورت دستی بر خلاف کد های پایتونی باید بنویسیم.

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

## ۷. زیرساخت مشترک خطاها — `pkg/errors`

قبل از نوشتن لایه‌ی repository، یک بار برای همیشه یک سیستم طبقه‌بندی خطا ساختیم که کل پروژه (نه فقط User Service) ازش استفاده می‌کنه:

- ✅ ساخت `pkg/errors/errors.go` با:
  - `type Kind int` و ثابت‌های `KindNotFound`, `KindAlreadyExists`, `KindInvalidInput`, `KindUnauthenticated`, `KindPermissionDenied`, `KindInternal`, `KindUnknown`
  - `type Error struct` که یک `Kind` + پیام + خطای زیرین که یک پارامتر اختیاری است رو نگه می‌داره
  - `New(kind, message)` برای ساخت خطای sentinel
  - `Wrap(kind, err, message)` برای پیچیدن یک خطای خام (مثلاً از pgx) با یک Kind
  - `GetKind(err)` برای این‌که لایه‌ی handler بتونه بدون شناختن خطای دقیق، فقط بر اساس `Kind` تصمیم بگیره کد gRPC مناسب چیه (مثلاً `KindNotFound` → `codes.NotFound`)

## ۸. پیاده‌سازی لایه `repository`

- ✅ ابتدا یک کتابخانه سریع و کامل برای ارتباط به دیتابیس نیاز داریم و من pgx رو انتخاب کردم و باید با دستور زیر نصبش کنیم :

<div dir="ltr">

```bash
  go get github.com/jackc/pgx/v5/pgxpool
```

</div>

- ✅ ساخت `pkg/postgres/postgres.go` (مشترک بین همه‌ی سرویس‌ها):
  - `NewPool(ctx, dsn) (*pgxpool.Pool, error)` — فقط یک DSN خام می‌گیره، هیچ وابستگی به دامنه‌ی خاصی نداره؛ منتقل‌شده از یک فایل محلی داخل `user-service` به `pkg` چون هر سرویسی که Postgres داره (Product, Order, Payment) دقیقاً همین boilerplate رو نیاز داره.
  - `type DBTX interface { Exec(...); Query(...); QueryRow(...) }` — زیرمجموعه‌ی مشترک متدهایی که هم `*pgxpool.Pool` و هم `pgx.Tx` پیاده‌سازی می‌کنن.
  - `WithTx(ctx, pool, fn)` — یک تابع دلخواه رو داخل یک تراکنش اجرا می‌کنه (commit در موفقیت، rollback در خطا).

  **چرا `DBTX` به‌جای `*pgxpool.Pool` مستقیم؟** اگر repository مستقیماً نوع `*pgxpool.Pool` رو تایپ کنه، هیچ‌وقت نمی‌شه چند عملیات از چند repository مختلف رو در یک تراکنش اتمیک انجام داد (مثلاً الگوی Transactional Outbox که در Order Service به‌خاطر Saga حتماً لازم می‌شه: نوشتن سفارش + نوشتن رویداد در جدول outbox، هر دو در یک تراکنش). با `DBTX`، همون repository هم با pool معمولی کار می‌کنه هم با یک `pgx.Tx`، بدون تغییر حتی یک خط کوئری.

- ✅ حالا فایل های مورد نیاز برای مدیریت لایه رپوزیتوری رو میسازیم :

<div dir="ltr">

```bash
  services/user-service/internal/repository/
  ├── user_repository.go            ← CRUD on users
  └── refresh_token_repository.go   ← refresh_tokens actions
```

</div>

## ۹. پیاده‌سازی `config`

- ✅ ساخت `services/user-service/config/config.go` با ساختارهای جدا برای `PostgresConfig`, `RedisConfig`, `JWTConfig` (به‌جای یک struct تخت با همه‌ی فیلدها با هم).
- ✅ الگوی **fail-fast** برای مقادیر حساس: `DB_PASSWORD` و `JWT_SECRET` اگر در env نباشن، سرویس همون ابتدا با خطای واضح متوقف می‌شه (نه این‌که با یک secret خالی یا پیش‌فرض ناامن ادامه بده).
- ✅ بقیه‌ی مقادیر (پورت، آدرس Redis، TTL توکن‌ها) fallback منطقی دارن چون نبودشون خطرناک نیست.
- ✅ متد `PostgresConfig.DSN()` رشته‌ی اتصال Postgres رو می‌سازه تا فرمت connection string فقط یک‌جا زندگی کنه.
- ✅ ساخت `.env.example` با همه‌ی متغیرهای لازم:

<div dir="ltr">

```bash
DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE
GRPC_PORT
REDIS_ADDR, REDIS_PASSWORD, REDIS_DB
JWT_SECRET, JWT_ACCESS_TTL, JWT_REFRESH_TTL

```

</div>

- برای ساخت یک رمز عبور امن و مناسب برای `JWT_SECRET` و `DB_PASSWORD`با استفاده از دستور زیر در ترمینال میتوانیم یک رمز عبور امن بگیریم:

<div dir="ltr">

```bash
  openssl rand -hex 32
```

  </div>

## ۱۰. پیاده‌سازی لایه `service`

این لایه خیلی بزرگه و در عین حال بین قسمت های مختلف پروژه هم کد های پراکنده داره و شامل قسمت های زیر می باشد :

1. هش کردن پسورد با bcrypt
2. تولید و بررسی توکن های JWT
3. تولید و بررسی کد های OTP
4. پیاده سازی متد های `Register`, `Login`, `VerifyOTP`, `RefreshToken`, `GetUser`, `Logout`

### هش کردن پسورد

- ✅ اول وارد ماژول pkg میشیم و یک پکیج مشترک برای هش کردن رمز عبور میسازیم که در سرویس های دیگر هم قابل استفاده باشد.

<div dir="ltr">

```bash
  cd pkg
  go get golang.org/x/crypto/bcrypt
```

</div>

در این فایل بوسیله پکیچ bcript باید دو تابع بسازیم که یکی برای تبدیل رشته رمز عبور به یک رشته هش شده و یکی برای مقایسه رشته رمز عبور دریافتی از کاربر و مقدار دخیره شده در دیتابیس است که خروجی true یا false میدهد.

### OTP

- ✅ ابتدا پکیج های ارتباط با ردیس و ساخت شانسه یکتای UUID را باید به پروژه اضافه کنیم.

<div dir="ltr">

```

cd services/user-service
go get github.com/redis/go-redis/v9
go get github.com/google/uuid

```

</div>

- ✅ در مرحله بعد داخل ماژول pkg یک فایل `pkg/auth/otp.go` برای ساخت شناسه OTP رندوم و غیر قابل پیش بینی بوسیله crypto/rand میسازیم تا بوسیله منبع تصادفی سیستم عامل شناسه های غیر قابل پیش بینی بسازد.

- ✅ سپس یک فایل `services/user-service/internal/model/otp.go` برای ذخیره دیتای هر چالش فعال در ردیس میسازیم.

- ✅ حالا نیاز به یک کلاینت برای اتصال به ردیس داریم برای این منظور فایل `services/user-service/internal/repository/redis.go` را ایجاد کرده و در آن تابع کلاینت ردیس را ایجاد میکنیم.

### توکن‌ها — JWT (Access Token) و Refresh Token

- ✅ ابتدا وارد پوشه `pkg` میشیم و کتابخانه مربوط به `JWT` رو نصب میکنیم:

<div dir="ltr">

```bash
  cd pkg
  go get github.com/golang-jwt/jwt/v5
```

</div>

- ✅ `pkg/auth/access_token.go`: تعریف `AccessClaims` (شامل `user_id` و `role` — role عمداً داخل claims هست چون طبق `architecture.md`، بررسی RBAC باید در سطح هر سرویس و بر اساس claims انجام بشه، بدون کوئری اضافه به دیتابیس)، `GenerateAccessToken`, `ParseAccessToken`.
  - نکته‌ی امنیتی مهم: `ParseAccessToken` صراحتاً چک می‌کنه الگوریتم امضا `HMAC` باشه، تا جلوی حمله‌ی "alg confusion" رو بگیره (یه مهاجم نتونه هدر توکن رو دستکاری کنه و الگوریتم امضا رو عوض کنه).
- ✅ `pkg/auth/refresh_token.go`: `GenerateRefreshToken` (یک رشته‌ی رندوم opaque با ۲۵۶ بیت آنتروپی، **نه یک JWT** — چون باید بشه قبل از انقضای طبیعیش باطلش کرد، که فقط با نگه‌داشتن در دیتابیس ممکنه) و `HashRefreshToken` (با SHA-256، نه bcrypt، چون خود توکن از قبل پرآنتروپیه و نیازی به کندی عمدی bcrypt نداره).
- ✅ `pkg/auth/token_manager.go`: `TokenManager` interface + پیاده‌سازیش که چهار تابع بالا رو پشت یک abstraction تزریق‌پذیر جمع می‌کنه (شامل `AccessTokenTTL()` هم برای این‌که `service` بتونه `expires_in` رو محاسبه کنه). به‌جای این‌که `UserService` مستقیم `secret` خام نگه داره و هر بار توابع سطح-پکیج رو صدا بزنه، فقط یک `auth.TokenManager` تزریق می‌شه — دقیقاً همون الگوی interface+constructor که برای repository ها استفاده شده.

### پیاده‌سازی متدهای اصلی سرویس

- ✅ `services/user-service/internal/service/user_service.go`:
  - `issueTokens` (خصوصی): یک تابع مشترک که access+refresh token می‌سازه و refresh token هش‌شده رو در دیتابیس ذخیره می‌کنه — چون `VerifyOTP` و `RefreshToken` هر دو دقیقاً همین کار رو نیاز دارن و تکرارش خطرناکه.
  - `Register`: هش‌کردن پسورد + ساخت کاربر با نقش پیش‌فرض `RoleMember`.
  - `OTPLogin`: چک پسورد → صدور OTP → ذخیره‌ی چالش در Redis.
  - `VerifyOTP`: چک کد OTP → حذف چالش (جلوگیری از replay) → صدور access/refresh token واقعی.
  - `RefreshToken`: پیاده‌سازی **Refresh Token Rotation** — هر بار refresh، توکن قدیمی `Revoke` می‌شه و یک توکن جدید صادر می‌شه؛ یعنی هر refresh token فقط یک‌بار مصرفه (اگر یه توکن باطل‌شده دوباره استفاده بشه، نشونه‌ی احتمالی سرقت توکنه).
  - `GetUser`: خواندن ساده‌ی کاربر بر اساس ID.
  - `Logout`: **idempotent** — یعنی اگه توکن از قبل پیدا نشه (مثلاً کاربر قبلاً logout کرده)، خطا برنمی‌گردونه؛ نتیجه از دید کلاینت یکیه.
- ✅ `services/user-service/internal/service/mapper.go`: تبدیل مدل داخلی `User` به `pb.User` (پیام proto)، جایی که فیلدهای حساس (`PasswordHash`) عمداً map نمی‌شن.

## ۱۱. سیم‌کشی در `main.go` (وضعیت فعلی، ناقص)

- ✅ `cmd/main.go` الان این‌ها رو به هم وصل می‌کنه:
  1. `config.Load()` برای خوندن env
  2. `postgres.NewPool(ctx, cfg.Postgres.DSN())` برای ساخت pool
  3. `repository.NewRedisClient(...)` برای ساخت کلاینت Redis
  4. ساخت `userRepo`, `refreshTokenRepo` (با پاس دادن مستقیم `pool` که `postgres.DBTX` رو پیاده‌سازی می‌کنه)، `otpRepo`
  5. `auth.NewTokenManager(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL)`
  6. `service.NewUserService(...)` با تزریق همه‌ی وابستگی‌های بالا
- ⏳ فعلاً `userService` ساخته می‌شه ولی جایی استفاده نمی‌شه (`_ = userService`) چون هنوز سرور gRPC (لایه‌ی `handler`) وصل نشده.

## ۱۲. فایل کانفیگ و اضافه کردن متغیر های محیطی

با این ساختار فعلی فایل کانفیگ در صورت لود شدن فایل `.env` در داکر به صورت خودکار این فایل کار خواهد کرد ولی برای اجرای سرویس در محیط لوکال نیاز است که این فایل در برنامه لود شود برای این کار پکیج مخصوص را در `pkg` لود میکنیم:

<div dir="ltr">

```bash
  cd pkg
  go get github.com/joho/godotenv
```

</div>

## مراحل بعدی (هنوز انجام نشده)

- ⏳ پیاده‌سازی لایه‌ی `handler`: پیاده‌سازی interface سرور gRPC تولیدشده از proto (`pb.UnimplementedUserServiceServer`)، که متدهای `UserService` رو صدا می‌زنه و خطاهای دامنه‌ای (با `appErrors.GetKind`) رو به کدهای استاندارد gRPC (`codes.NotFound`, `codes.Unauthenticated`, ...) تبدیل می‌کنه.
- ⏳ تکمیل `cmd/main.go`: ساخت واقعی `grpc.NewServer()`، ثبت `handler` روش، و `Listen` روی `cfg.GRPCPort`.
- ⏳ نوشتن Dockerfile واقعی برای این سرویس (فعلاً خالیه).
- ⏳ اضافه‌کردن middleware احراز هویت/logging در سطح gRPC (اگه لازم باشه، جدا از چیزی که در سطح Gateway انجام می‌شه).

</div>
