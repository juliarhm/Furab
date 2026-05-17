// Package client provides integration clients for pricing-service.
package client

import (
	"context"
	"errors"
)

var ErrDistanceNotAvailable = errors.New("distance not available")

type LocationClient interface {
	GetDeliveryDistance(ctx context.Context, orderID string) (float64, error)
}

type dummyLocationClient struct{}

func NewDummyLocationClient() LocationClient {
	return &dummyLocationClient{}
}

func (d *dummyLocationClient) GetDeliveryDistance(ctx context.Context, orderID string) (float64, error) {
	if orderID == "" {
		return 0, ErrDistanceNotAvailable
	}

	return 5.2, nil
}
