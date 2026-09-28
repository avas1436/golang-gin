// services/product-service/internal/service/mapper.go

package service

import (
	pb "pkg/proto/product"
	"product-service/internal/model"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// toProtoProduct مدل داخلی را به پیام gRPC تبدیل می‌کند
func toProtoProduct(p *model.Product) *pb.Product {

	// بررسی خالی نبودن داده ورودی
	if p == nil {
		return nil
	}

	return &pb.Product{
		Id:            p.ID.String(),
		Name:          p.Name,
		Description:   p.Description,
		Category:      p.Category,
		Price:         p.Price,
		TotalStock:    p.TotalStock,
		ReservedStock: p.ReservedStock,
		IsActive:      p.IsActive,
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
	}
}

// toProtoProductList چند محصول را یک‌جا تبدیل می‌کند؛ برای پاسخ Search
func toProtoProductList(products []*model.Product) []*pb.Product {

	result := make([]*pb.Product, 0, len(products))

	for _, p := range products {
		// جلوگیری از قرار گرفتن داده خالی در لیست
		if protoProduct := toProtoProduct(p); protoProduct != nil {
			result = append(result, protoProduct)
		}
	}

	return result
}
