// Package service implements the business logic for location-service.
package service

import (
	"context"
	"errors"

	"furab-backend/services/location-service/internal/model"
	"furab-backend/services/location-service/internal/repository"
)

// DriverServiceClient defines the interface for communicating with DriverService.
type DriverServiceClient interface {
	ValidateDriver(ctx context.Context, driverID string) (bool, error)
}

// LocationService defines the interface for location-service business logic.
type LocationService interface {
	UpdateDriverLocation(ctx context.Context, req model.UpdateLocationRequest) error
	UpdateDriverStatus(ctx context.Context, req model.UpdateStatusRequest) error
	FindNearbyDrivers(ctx context.Context, req model.SearchDriverRequest) ([]model.DriverLocationResponse, error)
	GetDriverLocation(ctx context.Context, driverID, orderID string) (*model.TrackLocationResponse, error)
	GetEmergencyLocation(ctx context.Context, driverID, orderID, actorType string) (*model.TrackLocationResponse, error)
}

// locationServiceImpl is the concrete implementation of LocationService.
type locationServiceImpl struct {
	repo         repository.LocationRepository
	driverClient DriverServiceClient
}

// NewLocationService creates a new LocationService.
func NewLocationService(repo repository.LocationRepository, driverClient DriverServiceClient) LocationService {
	return &locationServiceImpl{
		repo:         repo,
		driverClient: driverClient,
	}
}

func (s *locationServiceImpl) UpdateDriverLocation(ctx context.Context, req model.UpdateLocationRequest) error {
	if req.DriverID == "" {
		return errors.New("driver_id is required")
	}

	valid, err := s.driverClient.ValidateDriver(ctx, req.DriverID)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid driver")
	}

	return s.repo.UpdateLocation(ctx, req)
}

func (s *locationServiceImpl) UpdateDriverStatus(ctx context.Context, req model.UpdateStatusRequest) error {
	if req.DriverID == "" {
		return errors.New("driver_id is required")
	}

	return s.repo.UpdateStatus(ctx, req)
}

func (s *locationServiceImpl) FindNearbyDrivers(ctx context.Context, req model.SearchDriverRequest) ([]model.DriverLocationResponse, error) {
	if req.Radius <= 0 {
		return nil, errors.New("invalid radius")
	}

	geos, err := s.repo.SearchNearbyDrivers(ctx, req)
	if err != nil {
		return nil, err
	}

	var results []model.DriverLocationResponse
	for _, geo := range geos {
		driverID := geo.Name

		isActive, err := s.repo.IsDriverActive(ctx, driverID)
		if err != nil || !isActive {
			continue // skip inactive drivers
		}

		status, err := s.repo.GetStatus(ctx, driverID)
		if err != nil || status != "available" {
			continue // skip non-available drivers
		}

		results = append(results, model.DriverLocationResponse{
			DriverID:     driverID,
			Longitude:    geo.Longitude,
			Latitude:     geo.Latitude,
			Distance:     geo.Dist,
			DriverStatus: status,
		})
	}

	if results == nil {
		return []model.DriverLocationResponse{}, nil
	}

	return results, nil
}

func (s *locationServiceImpl) GetDriverLocation(ctx context.Context, driverID, orderID string) (*model.TrackLocationResponse, error) {
	if driverID == "" {
		return nil, errors.New("driver_id is required")
	}

	res, err := s.repo.TrackDriver(ctx, driverID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *locationServiceImpl) GetEmergencyLocation(ctx context.Context, driverID, orderID, actorType string) (*model.TrackLocationResponse, error) {
	if driverID == "" {
		return nil, errors.New("driver_id is required")
	}

	res, err := s.repo.TrackDriver(ctx, driverID)
	if err != nil {
		return nil, err
	}

	return res, nil
}
