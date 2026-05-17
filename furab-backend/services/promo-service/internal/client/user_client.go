// Package client provides integration clients for promo-service.
package client

import (
	"context"
	"errors"
)

var ErrUserNotEligibleForPromo = errors.New("user is not eligible for promo")

type UserClient interface {
	ValidateUserPromo(ctx context.Context, userID, promoCode string) (bool, error)
}

type dummyUserClient struct{}

func NewDummyUserClient() UserClient {
	return &dummyUserClient{}
}

func (*dummyUserClient) ValidateUserPromo(ctx context.Context, userID, promoCode string) (bool, error) {
	if userID == "" {
		return false, ErrUserNotEligibleForPromo
	}
	return true, nil
}
