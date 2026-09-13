// services/order-service/internal/service/mapper.go

package service

import (
	pb "pkg/proto/order"

	"order-service/internal/model"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// toProtoOrder مدل داخلی را به پیام gRPC تبدیل می‌کند.
// اگر order.Items خالی باشد (مثلاً نتیجه‌ی ListByUser که عمداً
// آیتم‌ها را بار نمی‌کند)، فیلد Items پیام هم به‌سادگی خالی
// می‌ماند — این دقیقاً همان رفتاری است که در order.proto مستند شده
func toProtoOrder(o *model.Order) *pb.Order {

	return &pb.Order{
		Id:          o.ID.String(),
		UserId:      o.UserID.String(),
		Status:      string(o.Status),
		TotalAmount: o.TotalAmount,
		Items:       toProtoOrderItems(o.Items),
		CreatedAt:   timestamppb.New(o.CreatedAt),
		UpdatedAt:   timestamppb.New(o.UpdatedAt),
	}
}

func toProtoOrderItems(items []*model.OrderItem) []*pb.OrderItem {

	result := make([]*pb.OrderItem, 0, len(items))

	for _, item := range items {
		result = append(result, &pb.OrderItem{
			ProductId:   item.ProductID.String(),
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			Subtotal:    item.Subtotal,
		})
	}

	return result
}

func toProtoOrderList(orders []*model.Order) []*pb.Order {

	result := make([]*pb.Order, 0, len(orders))

	for _, o := range orders {
		result = append(result, toProtoOrder(o))
	}

	return result
}
