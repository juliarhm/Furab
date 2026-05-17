// Package repository provides data access layer for location-service.
package repository

import (
	"context"
	"fmt"
	"time"

	"furab-backend/services/location-service/internal/model"

	"github.com/redis/go-redis/v9"
)

// LocationRepository defines the interface for location-service data access.
type LocationRepository interface {
	UpdateLocation(ctx context.Context, req model.UpdateLocationRequest) error
	UpdateStatus(ctx context.Context, req model.UpdateStatusRequest) error
	GetStatus(ctx context.Context, driverID string) (string, error)
	SearchNearbyDrivers(ctx context.Context, req model.SearchDriverRequest) ([]redis.GeoLocation, error)
	TrackDriver(ctx context.Context, driverID string) (*model.TrackLocationResponse, error)
	IsDriverActive(ctx context.Context, driverID string) (bool, error)
}

// redisLocationRepository implements LocationRepository using Redis.
type redisLocationRepository struct {
	rdb *redis.Client
}

// NewRedisLocationRepository creates a new Redis-based repository.
func NewRedisLocationRepository(rdb *redis.Client) LocationRepository {
	return &redisLocationRepository{
		rdb: rdb,
	}
}

func (r *redisLocationRepository) UpdateLocation(ctx context.Context, req model.UpdateLocationRequest) error {
	err := r.rdb.GeoAdd(ctx, "driver_locations", &redis.GeoLocation{
		Name:      req.DriverID,
		Longitude: req.Longitude,
		Latitude:  req.Latitude,
	}).Err()
	if err != nil {
		return err
	}

	ttlKey := "driver_active:" + req.DriverID
	return r.rdb.Set(ctx, ttlKey, req.Timestamp.Unix(), 5*time.Minute).Err()
}

func (r *redisLocationRepository) UpdateStatus(ctx context.Context, req model.UpdateStatusRequest) error {
	statusKey := "driver_status:" + req.DriverID
	return r.rdb.Set(ctx, statusKey, req.DriverStatus, 0).Err() // Status persists until changed
}

func (r *redisLocationRepository) GetStatus(ctx context.Context, driverID string) (string, error) {
	statusKey := "driver_status:" + driverID
	status, err := r.rdb.Get(ctx, statusKey).Result()
	if err == redis.Nil {
		return "available", nil // default if not set
	}
	return status, err
}

func (r *redisLocationRepository) IsDriverActive(ctx context.Context, driverID string) (bool, error) {
	ttlKey := "driver_active:" + driverID
	res, err := r.rdb.Exists(ctx, ttlKey).Result()
	if err != nil {
		return false, err
	}
	return res > 0, nil
}

func (r *redisLocationRepository) SearchNearbyDrivers(ctx context.Context, req model.SearchDriverRequest) ([]redis.GeoLocation, error) {
	return r.rdb.GeoSearchLocation(ctx, "driver_locations", &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  req.LongitudeOrigin,
			Latitude:   req.LatitudeOrigin,
			Radius:     req.Radius,
			RadiusUnit: "km",
			Sort:       "ASC",
		},
		WithCoord: true,
		WithDist:  true,
	}).Result()
}

func (r *redisLocationRepository) TrackDriver(ctx context.Context, driverID string) (*model.TrackLocationResponse, error) {
	res, err := r.rdb.GeoPos(ctx, "driver_locations", driverID).Result()
	if err != nil {
		return nil, err
	}
	if len(res) == 0 || res[0] == nil {
		return nil, fmt.Errorf("driver location not found")
	}

	// Try to get timestamp
	ttlKey := "driver_active:" + driverID
	timestampStr, _ := r.rdb.Get(ctx, ttlKey).Result()
	var ts time.Time
	if timestampStr != "" {
		var unixTime int64
		fmt.Sscanf(timestampStr, "%d", &unixTime)
		ts = time.Unix(unixTime, 0)
	} else {
		ts = time.Now()
	}

	return &model.TrackLocationResponse{
		DriverID:  driverID,
		Longitude: res[0].Longitude,
		Latitude:  res[0].Latitude,
		Timestamp: ts,
	}, nil
}
