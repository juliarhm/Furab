// Package service implements the business logic for driver-service.
package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"furab-backend/services/driver-service/internal/model"
	"furab-backend/services/driver-service/internal/repository"
)

// Common service errors.
var (
	ErrInvalidRequest  = errors.New("invalid request")
	ErrDriverNotFound  = errors.New("driver not found")
	ErrStatusInvalid   = errors.New("invalid driver status")
	ErrLocationInvalid = errors.New("invalid location coordinates")
)

// DriverService defines the interface for driver-service business logic.
type DriverService interface {
	// CreateDriver registers a new driver.
	CreateDriver(ctx context.Context, req *model.CreateDriverRequest) (*model.DriverResponse, error)

	// GetDriver retrieves a driver by ID.
	GetDriver(ctx context.Context, driverID string) (*model.Driver, error)

	// UpdateDriver updates driver profile data.
	UpdateDriver(ctx context.Context, driverID string, req *model.UpdateDriverRequest) (*model.DriverResponse, error)

	// UpdateStatus changes a driver's availability status.
	UpdateStatus(ctx context.Context, driverID, status string) (*model.DriverResponse, error)

	// UpdateLocation updates a driver's GPS coordinates.
	UpdateLocation(ctx context.Context, driverID string, lat, long float64) (*model.DriverResponse, error)
}

// driverServiceImpl is the concrete implementation of DriverService.
type driverServiceImpl struct {
	repo repository.DriverRepository
}

// NewDriverService creates a new DriverService.
func NewDriverService(repo repository.DriverRepository) DriverService {
	return &driverServiceImpl{repo: repo}
}

// CreateDriver registers a new driver.
func (s *driverServiceImpl) CreateDriver(ctx context.Context, req *model.CreateDriverRequest) (*model.DriverResponse, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}

	// Normalize input
	req.DriverID = strings.TrimSpace(req.DriverID)
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.VehicleType = strings.TrimSpace(req.VehicleType)

	if err := req.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	driver := &model.Driver{
		DriverID:    req.DriverID,
		Name:        req.Name,
		Phone:       req.Phone,
		VehicleType: req.VehicleType,
		Status:      model.DriverStatusOffline,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Save(ctx, driver); err != nil {
		return nil, err
	}

	return &model.DriverResponse{
		Status:   "success",
		Message:  "driver created successfully",
		DriverID: driver.DriverID,
	}, nil
}

// GetDriver retrieves a driver by ID.
func (s *driverServiceImpl) GetDriver(ctx context.Context, driverID string) (*model.Driver, error) {
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return nil, ErrInvalidRequest
	}

	driver, err := s.repo.FindByID(ctx, driverID)
	if err != nil {
		if errors.Is(err, repository.ErrDriverNotFound) {
			return nil, ErrDriverNotFound
		}
		return nil, err
	}

	return driver, nil
}

// UpdateDriver updates driver profile data.
func (s *driverServiceImpl) UpdateDriver(ctx context.Context, driverID string, req *model.UpdateDriverRequest) (*model.DriverResponse, error) {
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return nil, ErrInvalidRequest
	}

	if req == nil {
		return nil, ErrInvalidRequest
	}

	// Normalize input
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.VehicleType = strings.TrimSpace(req.VehicleType)

	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get existing driver
	driver, err := s.repo.FindByID(ctx, driverID)
	if err != nil {
		if errors.Is(err, repository.ErrDriverNotFound) {
			return nil, ErrDriverNotFound
		}
		return nil, err
	}

	// Update fields
	driver.Name = req.Name
	driver.Phone = req.Phone
	driver.VehicleType = req.VehicleType
	driver.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, driver); err != nil {
		return nil, err
	}

	return &model.DriverResponse{
		Status:  "success",
		Message: "driver updated successfully",
	}, nil
}

// UpdateStatus changes a driver's availability status.
func (s *driverServiceImpl) UpdateStatus(ctx context.Context, driverID, status string) (*model.DriverResponse, error) {
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return nil, ErrInvalidRequest
	}

	// Validate and normalize status
	normalizedStatus := model.DriverStatus(strings.ToUpper(strings.TrimSpace(status)))
	if !normalizedStatus.IsValid() {
		return nil, ErrStatusInvalid
	}

	// Check driver exists
	if _, err := s.repo.FindByID(ctx, driverID); err != nil {
		if errors.Is(err, repository.ErrDriverNotFound) {
			return nil, ErrDriverNotFound
		}
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, driverID, string(normalizedStatus)); err != nil {
		return nil, err
	}

	return &model.DriverResponse{
		Status:  "success",
		Message: "status updated successfully",
	}, nil
}

// UpdateLocation updates a driver's GPS coordinates.
func (s *driverServiceImpl) UpdateLocation(ctx context.Context, driverID string, lat, long float64) (*model.DriverResponse, error) {
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return nil, ErrInvalidRequest
	}

	// Validate coordinates
	if err := validateLocation(lat, long); err != nil {
		return nil, err
	}

	// Check driver exists
	if _, err := s.repo.FindByID(ctx, driverID); err != nil {
		if errors.Is(err, repository.ErrDriverNotFound) {
			return nil, ErrDriverNotFound
		}
		return nil, err
	}

	if err := s.repo.UpdateLocation(ctx, driverID, lat, long); err != nil {
		return nil, err
	}

	return &model.DriverResponse{
		Status:  "success",
		Message: "location updated successfully",
	}, nil
}

// validateLocation validates GPS coordinates.
func validateLocation(lat, long float64) error {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(long) || math.IsInf(long, 0) {
		return ErrLocationInvalid
	}
	if lat < -90 || lat > 90 || long < -180 || long > 180 {
		return ErrLocationInvalid
	}
	return nil
}
