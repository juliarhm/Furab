// Package client provides integration clients for promo-service.
package client

import (
	"context"
	"errors"
)

var ErrOrderNotValidForPromo = errors.New("order does not satisfy promo requirements")

type OrderClient interface {
	ValidateOrderPromo(ctx context.Context, orderID, promoCode string) (bool, error)
}

type dummyOrderClient struct{}

func NewDummyOrderClient() OrderClient {
	return &dummyOrderClient{}
}

func (*dummyOrderClient) ValidateOrderPromo(ctx context.Context, orderID, promoCode string) (bool, error) {
	if orderID == "" {
		return false, ErrOrderNotValidForPromo
	}
	return true, nil
}
