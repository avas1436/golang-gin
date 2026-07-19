<div dir="rtl">

# 🛍️ فروشگاه آنلاین با معماری میکروسرویس

> پروژه‌ای تمرینی برای یادگیری عمیق معماری میکروسرویس با Go و Gin

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![Gin](https://img.shields.io/badge/Gin-Framework-00ADD8)](https://gin-gonic.com)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## درباره پروژه

این یک پروژه تمرینی برای ساخت بک‌اند یک فروشگاه آنلاین با معماری میکروسرویس است که با Go و فریم‌ورک Gin توسعه داده می‌شود. هدف اصلی این پروژه، یادگیری عملی مفاهیم طراحی سیستم‌های توزیع‌شده و معماری میکروسرویس است؛ به همین دلیل تمرکز بر روی **الگوهای معماری و تصمیمات طراحی** است، نه بهینه‌سازی برای محیط production یا مقیاس‌پذیری واقعی.

مفاهیم و قابلیت‌هایی که در این پروژه پیاده‌سازی می‌شوند:

- مسیر یابی متمرکز درخواست در API Gateway
- احراز هویت با JWT و Refresh Token
- پیاده سازی Event-Driven Architecture با استفاده از Message Broker
- الگوی Saga برای مدیریت تراکنش‌های توزیع‌شده
- پیاده سازی Rate Limiting برای کنترل بار درخواست‌ها
- کش کردن با Redis
- کنترل دسترسی مبتنی بر نقش (RBAC: Admin, Member, Viewer)
- پردازش‌های ناهمگام (Task/Job Queue)
- احراز هویت با OTP و رمز عبور
- جست‌وجوی محصولات
- تست‌های واحد (Unit Test)
- پایپ‌لاین CI/CD

### تکنولوژی‌های استفاده‌شده

<div dir="ltr">

| Category                    | Technology               |
| --------------------------- | ------------------------ |
| Programming Language        | Go                       |
| Web Framework               | Gin                      |
| Inter-service Communication | REST, gRPC               |
| Async Messaging             | RabbitMQ                 |
| Database                    | PostgreSQL (per service) |
| Cache                       | Redis                    |
| Authentication              | JWT, Refresh Token, OTP  |
| Containerization            | Docker, Docker Compose   |
| CI/CD                       | GitHub Actions           |
| API Documentation           | OpenAPI/Swagger          |

</div>

## معماری

این پروژه از معماری میکروسرویس با یک API Gateway مرکزی استفاده می‌کند. ارتباط بین سرویس‌ها بخشی به صورت همزمان (REST/gRPC) و بخشی به صورت ناهمزمان (از طریق RabbitMQ) پیاده‌سازی شده تا سناریوهای واقعی سیستم‌های توزیع‌شده تجربه شوند.

برای مشاهده جزئیات کامل معماری، دیاگرام‌ها و جریان داده بین سرویس‌ها به فایل [docs/architecture.md](docs/architecture.md) مراجعه کنید.

برای مشاهده دلایل و پیامدهای تصمیمات معماری گرفته‌شده در این پروژه، به [docs/decisions](docs/decisions) مراجعه کنید.

## سرویس‌ها

<div dir="ltr">

| Service Name         | Responsibility                                  | Port | Specific Technologies   |
| -------------------- | ----------------------------------------------- | ---- | ----------------------- |
| API Gateway          | Request routing, authentication, Rate Limiting  | 8080 | Gin, Middleware         |
| User Service         | Registration, login, user management, JWT & OTP | 8081 | JWT, Redis              |
| Product Service      | Product management and search                   | 8082 | PostgreSQL, Redis Cache |
| Order Service        | Order placement and tracking, Saga pattern      | 8083 | RabbitMQ                |
| Payment Service      | Bank gateway simulation                         | 8084 | RabbitMQ                |
| Notification Service | SMS/Email sending based on events               | 8085 | RabbitMQ                |

</div>

## پیش‌نیازها

- Go نسخه 1.26 یا بالاتر
- Docker و Docker Compose
- Make

## نحوه اجرا

راهنمای کامل نصب، اجرا و troubleshooting در فایل [docs/setup.md](docs/setup.md) موجود است.

## نقشه راه (Roadmap)

<div dir="ltr">

- [x] [Initial project structure setup](docs/decisions/001-initial-project-structure-setup.md)
- [ ] User Service implementation with JWT
- [ ] Product Service implementation
- [ ] Order Service implementation with Saga pattern
- [ ] RabbitMQ integration for asynchronous communication
- [ ] API Gateway implementation
- [ ] CI/CD pipeline setup

</div>

## لایسنس

این پروژه تحت لایسنس MIT منتشر شده است.

</div>
