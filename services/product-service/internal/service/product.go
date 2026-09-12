// services/product-service/internal/service/product.go

package service

import (
	"context"
	"pkg/auth"
	appErrors "pkg/errors"
	pb "pkg/proto/product"

	"product-service/internal/model"
	"product-service/internal/repository"

	"github.com/google/uuid"
)

// مقدار ثابت نقش ادمین
const roleAdmin = "admin"

// ساختار سرویس محصول
type ProductService struct {
	productRepo repository.ProductRepository
}

// سازنده یک ساختار سرویس محصول
func NewProductService(
	productRepo repository.ProductRepository,
) *ProductService {

	return &ProductService{
		productRepo: productRepo,
	}
}

// یک تابع مشترک برای چک کردن نقش کاربر
func requireAdmin(ctx context.Context) error {

	// دریافت مقادیر احراز هویت که در کانتکست قرار گرفته
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return appErrors.New(
			appErrors.KindUnauthenticated,
			"authentication required",
		)
	}

	// بررسی میکند نقش ادمین باشد
	if claims.Role != roleAdmin {
		return appErrors.New(
			appErrors.KindPermissionDenied,
			"only admins can perform this action",
		)
	}

	return nil
}

// کار این تابع اینه که تایپ رشته دریافت شده از سرویس های دیگر
// رو به تایپ uuid تبدیل میکنه
func parseProductID(id string) (uuid.UUID, error) {

	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, appErrors.New(
			appErrors.KindInvalidInput,
			"invalid product id",
		)
	}

	return parsed, nil
}

// ساخت ادمین برای کاربر ادمین
func (
	s *ProductService,
) CreateProduct(
	ctx context.Context,
	req *pb.CreateProductRequest,
) (
	*pb.Product, error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"create product request is nil",
		)
	}

	// بررسی نقش درخواست کننده
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	// ساخت شی اولیه محصول
	p := &model.Product{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Price:       req.Price,
		TotalStock:  req.TotalStock,
	}

	// ساخت محصول جدید و قرار دادن مقادیر خالی در شی اولیه
	if err := s.productRepo.Create(ctx, p); err != nil {
		return nil, err
	}

	// خروجی شی کامل شده
	return toProtoProduct(p), nil
}

// ویرایش مقادیر محصول فقط برای ادمین
func (
	s *ProductService,
) UpdateProduct(
	ctx context.Context,
	req *pb.UpdateProductRequest,
) (
	*pb.Product, error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"update product request is nil",
		)
	}

	// بررسی نقش درخواست کننده
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	// تبدیل آیدی محصول از رشته به UUID
	id, err := parseProductID(req.Id)
	if err != nil {
		return nil, err
	}

	// ساخت شی محصول از مقادیر درخواست
	p := &model.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Price:       req.Price,
		IsActive:    req.IsActive,
	}

	// آپدیت ویژگی های محصول
	if err := s.productRepo.Update(ctx, p); err != nil {
		return nil, err
	}

	// چون برای آپدیت رپوزیتوری تنها تاریخ آخرین آپدیت را در شی محصول
	// قرار میدهد مجدد مشخصات محصول را اضافه میکنیم
	fresh, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return toProtoProduct(fresh), nil
}

// دریافت مشخصات محصول
func (
	s *ProductService,
) GetProduct(
	ctx context.Context,
	req *pb.GetProductRequest,
) (
	*pb.Product, error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"get product request is nil",
		)
	}

	id, err := parseProductID(req.Id)
	if err != nil {
		return nil, err
	}

	p, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return toProtoProduct(p), nil
}

// جستجوی محصول که هم میتواند با کتگوری و هم با متن نام و توضیحات باشد
func (
	s *ProductService,
) SearchProducts(
	ctx context.Context,
	req *pb.SearchProductsRequest,
) (
	*pb.SearchProductsResponse, error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"search products request is nil",
		)
	}

	limit := int(req.Limit)
	offset := int(req.Offset)

	products, err := s.productRepo.Search(
		ctx,
		req.Query,
		req.Category,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}

	return &pb.SearchProductsResponse{
		Products: toProtoProductList(products),
	}, nil
}

// ReserveStock توسط Order Service به‌صورت sync فراخوانی می‌شود
// (مرحله‌ی اول Saga، قبل از انتشار order.created)
func (
	s *ProductService,
) ReserveStock(
	ctx context.Context,
	req *pb.ReserveStockRequest,
) (
	*pb.ReserveStockResponse, error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"reserve stock request is nil",
		)
	}

	id, err := parseProductID(req.ProductId)
	if err != nil {
		return nil, err
	}

	if err := s.productRepo.ReserveStock(
		ctx,
		id,
		req.Quantity,
	); err != nil {
		return nil, err
	}

	return &pb.ReserveStockResponse{}, nil
}

// ReleaseStock عملیات جبرانی Saga است (payment.failed)
func (
	s *ProductService,
) ReleaseStock(
	ctx context.Context,
	req *pb.ReleaseStockRequest,
) (
	*pb.ReleaseStockResponse, error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"release stock request is nil",
		)
	}

	id, err := parseProductID(req.ProductId)
	if err != nil {
		return nil, err
	}

	if err := s.productRepo.ReleaseStock(
		ctx,
		id,
		req.Quantity,
	); err != nil {
		return nil, err
	}

	return &pb.ReleaseStockResponse{}, nil
}

// ConfirmStock رزرو را به مصرف قطعی تبدیل می‌کند (payment.completed)
func (
	s *ProductService,
) ConfirmStock(
	ctx context.Context,
	req *pb.ConfirmStockRequest,
) (
	*pb.ConfirmStockResponse, error,
) {

	if req == nil {
		return nil, appErrors.New(
			appErrors.KindInvalidInput,
			"confirm stock request is nil",
		)
	}

	id, err := parseProductID(req.ProductId)
	if err != nil {
		return nil, err
	}

	if err := s.productRepo.ConfirmStock(
		ctx, id, req.Quantity,
	); err != nil {
		return nil, err
	}

	return &pb.ConfirmStockResponse{}, nil
}
