// services/payment-service/internal/service/mapper.go

package service

import (
	pb "pkg/proto/payment"

	"payment-service/internal/model"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// toProtoPayment مدل داخلی را به پیام gRPC تبدیل می‌کند.
// فیلدهای اشاره‌گر اختیاری (GatewayName/GatewayRefID/FailureReason)
// اگر nil باشند، به رشته‌ی خالی proto3 تبدیل می‌شوند
func toProtoPayment(p *model.Payment) *pb.Payment {

	meta, _ := structpb.NewStruct(p.Metadata)

	var gatewayName, gatewayRefID, failureReason string

	if p.GatewayName != nil {
		gatewayName = *p.GatewayName
	}
	if p.GatewayRefID != nil {
		gatewayRefID = *p.GatewayRefID
	}
	if p.FailureReason != nil {
		failureReason = *p.FailureReason
	}

	return &pb.Payment{
		Id:            p.ID.String(),
		OrderId:       p.OrderID.String(),
		UserId:        p.UserID.String(),
		Currency:      string(p.Currency),
		Amount:        p.Amount,
		Status:        string(p.Status),
		GatewayName:   gatewayName,
		GatewayRefId:  gatewayRefID,
		FailureReason: failureReason,
		Metadata:      meta,
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
	}
}
