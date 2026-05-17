// Package client provides integration clients for pricing-service.
package client

import (
	"context"
	"errors"
	"furab-backend/services/pricing-service/internal/model"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderClient interface {
	GetOrderItems(ctx context.Context, orderID string) ([]model.OrderItem, error)
}

type dummyOrderClient struct{}

func NewDummyOrderClient() OrderClient {
	return &dummyOrderClient{}
}

func (d *dummyOrderClient) GetOrderItems(ctx context.Context, orderID string) ([]model.OrderItem, error) {
	if orderID == "" {
		return nil, ErrOrderNotFound
	}

	return []model.OrderItem{
		{ProductID: "item-001", Quantity: 2, UnitPrice: 15000},
		{ProductID: "item-002", Quantity: 1, UnitPrice: 23000},
	}, nil
}
