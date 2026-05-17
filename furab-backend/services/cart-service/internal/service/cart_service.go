// Package service implements the business logic for cart-service.
package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/cart-service/internal/model"
	"furab-backend/services/cart-service/internal/repository"

	"github.com/google/uuid"
)

// Common service errors.
var (
	ErrInvalidRequest  = errors.New("invalid request")
	ErrCartNotFound    = errors.New("cart not found")
	ErrItemNotFound    = errors.New("item not found in cart")
	ErrCartEmpty       = errors.New("cart is empty")
	ErrDiffMerchant    = errors.New("cannot add items from different merchant")
)

// CartService defines the interface for cart business logic.
type CartService interface {
	// GetCart retrieves the cart for a user. Returns empty cart if none exists.
	GetCart(ctx context.Context, userID string) (*model.Cart, error)

	// AddItem adds an item to the user's cart. If the item already exists, increment quantity.
	AddItem(ctx context.Context, userID string, req *model.AddItemRequest) (*model.Cart, error)

	// UpdateItemQuantity updates the quantity of an item. If quantity is 0, removes the item.
	UpdateItemQuantity(ctx context.Context, userID, itemID string, quantity int) (*model.Cart, error)

	// RemoveItem removes an item from the cart.
	RemoveItem(ctx context.Context, userID, itemID string) (*model.Cart, error)

	// ClearCart removes all items from the cart.
	ClearCart(ctx context.Context, userID string) error

	// GetCartTotal returns the total price of the cart.
	GetCartTotal(ctx context.Context, userID string) (float64, error)
}

// cartServiceImpl is the concrete implementation of CartService.
type cartServiceImpl struct {
	repo repository.CartRepository
}

// NewCartService creates a new CartService with the given dependencies.
func NewCartService(repo repository.CartRepository) CartService {
	return &cartServiceImpl{repo: repo}
}

// GetCart retrieves the cart for a user.
func (s *cartServiceImpl) GetCart(ctx context.Context, userID string) (*model.Cart, error) {
	if userID == "" {
		return nil, ErrInvalidRequest
	}

	cart, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			// Return empty cart
			return &model.Cart{
				ID:     uuid.New().String(),
				UserID: userID,
				Items:  []model.CartItem{},
			}, nil
		}
		return nil, err
	}

	return cart, nil
}

// AddItem adds an item to the user's cart.
func (s *cartServiceImpl) AddItem(ctx context.Context, userID string, req *model.AddItemRequest) (*model.Cart, error) {
	if userID == "" {
		return nil, ErrInvalidRequest
	}
	if req == nil {
		return nil, ErrInvalidRequest
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get or create cart
	cart, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			cart = &model.Cart{
				ID:         uuid.New().String(),
				UserID:     userID,
				MerchantID: req.MerchantID,
				Items:      []model.CartItem{},
				CreatedAt:  time.Now().UTC(),
			}
		} else {
			return nil, err
		}
	}

	// Check merchant consistency (cart can only have items from one merchant)
	if cart.MerchantID != "" && cart.MerchantID != req.MerchantID {
		return nil, ErrDiffMerchant
	}

	// Check if item already exists → increment quantity
	if existing, idx := cart.FindItemByMenuID(req.MenuItemID); idx >= 0 {
		existing.Quantity += req.Quantity
		cart.Items[idx] = *existing
	} else {
		// Add new item
		cart.Items = append(cart.Items, model.CartItem{
			ID:         uuid.New().String(),
			MenuItemID: req.MenuItemID,
			MerchantID: req.MerchantID,
			Name:       req.Name,
			Price:      req.Price,
			Quantity:   req.Quantity,
			Notes:      req.Notes,
		})
	}

	cart.MerchantID = req.MerchantID
	cart.CalculateTotal()
	cart.UpdatedAt = time.Now().UTC()

	if err := s.repo.Save(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

// UpdateItemQuantity updates the quantity of an item in the cart.
func (s *cartServiceImpl) UpdateItemQuantity(ctx context.Context, userID, itemID string, quantity int) (*model.Cart, error) {
	if userID == "" || itemID == "" {
		return nil, ErrInvalidRequest
	}

	cart, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			return nil, ErrCartNotFound
		}
		return nil, err
	}

	_, idx := cart.FindItem(itemID)
	if idx < 0 {
		return nil, ErrItemNotFound
	}

	if quantity <= 0 {
		// Remove the item
		cart.Items = append(cart.Items[:idx], cart.Items[idx+1:]...)
	} else {
		cart.Items[idx].Quantity = quantity
	}

	cart.CalculateTotal()
	cart.UpdatedAt = time.Now().UTC()

	if err := s.repo.Save(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

// RemoveItem removes an item from the cart.
func (s *cartServiceImpl) RemoveItem(ctx context.Context, userID, itemID string) (*model.Cart, error) {
	if userID == "" || itemID == "" {
		return nil, ErrInvalidRequest
	}

	cart, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			return nil, ErrCartNotFound
		}
		return nil, err
	}

	_, idx := cart.FindItem(itemID)
	if idx < 0 {
		return nil, ErrItemNotFound
	}

	cart.Items = append(cart.Items[:idx], cart.Items[idx+1:]...)
	cart.CalculateTotal()
	cart.UpdatedAt = time.Now().UTC()

	if err := s.repo.Save(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

// ClearCart removes all items from the cart.
func (s *cartServiceImpl) ClearCart(ctx context.Context, userID string) error {
	if userID == "" {
		return ErrInvalidRequest
	}

	return s.repo.Delete(ctx, userID)
}

// GetCartTotal returns the total price of the cart.
func (s *cartServiceImpl) GetCartTotal(ctx context.Context, userID string) (float64, error) {
	if userID == "" {
		return 0, ErrInvalidRequest
	}

	cart, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			return 0, nil
		}
		return 0, err
	}

	cart.CalculateTotal()
	return cart.TotalPrice, nil
}
