package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"furab-backend/services/location-service/internal/model"
	"furab-backend/services/location-service/internal/service"
	"furab-backend/services/location-service/test/unit/mock"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUpdateDriverLocation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockLocationRepository(ctrl)
	mockDriverClient := mock.NewMockDriverServiceClient(ctrl)
	svc := service.NewLocationService(mockRepo, mockDriverClient)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		req := model.UpdateLocationRequest{
			DriverID:  "driver1",
			Latitude:  -6.200000,
			Longitude: 106.816666,
			Timestamp: time.Now(),
		}

		mockDriverClient.EXPECT().ValidateDriver(ctx, "driver1").Return(true, nil)
		mockRepo.EXPECT().UpdateLocation(ctx, req).Return(nil)

		err := svc.UpdateDriverLocation(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("Error - Driver ID kosong", func(t *testing.T) {
		req := model.UpdateLocationRequest{
			DriverID:  "",
			Latitude:  -6.200000,
			Longitude: 106.816666,
			Timestamp: time.Now(),
		}

		err := svc.UpdateDriverLocation(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "driver_id is required", err.Error())
	})

	t.Run("Error - Driver tidak valid", func(t *testing.T) {
		req := model.UpdateLocationRequest{
			DriverID:  "driver_invalid",
			Latitude:  -6.200000,
			Longitude: 106.816666,
			Timestamp: time.Now(),
		}

		mockDriverClient.EXPECT().ValidateDriver(ctx, "driver_invalid").Return(false, nil)

		err := svc.UpdateDriverLocation(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "invalid driver", err.Error())
	})

	t.Run("Error - Repository gagal", func(t *testing.T) {
		req := model.UpdateLocationRequest{
			DriverID:  "driver2",
			Latitude:  -6.200000,
			Longitude: 106.816666,
			Timestamp: time.Now(),
		}

		mockDriverClient.EXPECT().ValidateDriver(ctx, "driver2").Return(true, nil)
		mockRepo.EXPECT().UpdateLocation(ctx, req).Return(errors.New("db error"))

		err := svc.UpdateDriverLocation(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})
}

func TestUpdateDriverStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockLocationRepository(ctrl)
	mockDriverClient := mock.NewMockDriverServiceClient(ctrl)
	svc := service.NewLocationService(mockRepo, mockDriverClient)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		req := model.UpdateStatusRequest{
			DriverID:     "driver1",
			DriverStatus: "available",
		}

		mockRepo.EXPECT().UpdateStatus(ctx, req).Return(nil)

		err := svc.UpdateDriverStatus(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("Error - Driver ID kosong", func(t *testing.T) {
		req := model.UpdateStatusRequest{
			DriverID:     "",
			DriverStatus: "available",
		}

		err := svc.UpdateDriverStatus(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "driver_id is required", err.Error())
	})

	t.Run("Error - Repository gagal", func(t *testing.T) {
		req := model.UpdateStatusRequest{
			DriverID:     "driver1",
			DriverStatus: "available",
		}

		mockRepo.EXPECT().
			UpdateStatus(ctx, req).
			Return(errors.New("db error"))

		err := svc.UpdateDriverStatus(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})
}

func TestFindNearbyDrivers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockLocationRepository(ctrl)
	mockDriverClient := mock.NewMockDriverServiceClient(ctrl)
	svc := service.NewLocationService(mockRepo, mockDriverClient)

	ctx := context.Background()

	t.Run("Success - Return driver available saja", func(t *testing.T) {
		req := model.SearchDriverRequest{
			LatitudeOrigin:  -6.200000,
			LongitudeOrigin: 106.816666,
			Radius:          5,
		}

		mockGeos := []redis.GeoLocation{
			{Name: "driver1", Longitude: 106.816666, Latitude: -6.200000, Dist: 1.2},
			{Name: "driver2", Longitude: 106.816666, Latitude: -6.200000, Dist: 1.5},
			{Name: "driver3", Longitude: 106.816666, Latitude: -6.200000, Dist: 1.8},
		}

		mockRepo.EXPECT().SearchNearbyDrivers(ctx, req).Return(mockGeos, nil)

		// driver1 is available
		mockRepo.EXPECT().IsDriverActive(ctx, "driver1").Return(true, nil)
		mockRepo.EXPECT().GetStatus(ctx, "driver1").Return("available", nil)

		// driver2 is busy
		mockRepo.EXPECT().IsDriverActive(ctx, "driver2").Return(true, nil)
		mockRepo.EXPECT().GetStatus(ctx, "driver2").Return("busy", nil)

		// driver3 is inactive
		mockRepo.EXPECT().IsDriverActive(ctx, "driver3").Return(false, nil)

		drivers, err := svc.FindNearbyDrivers(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, drivers, 1)
		assert.Equal(t, "driver1", drivers[0].DriverID)
		assert.Equal(t, "available", drivers[0].DriverStatus)
	})

	t.Run("Success - Tidak ada driver", func(t *testing.T) {
		req := model.SearchDriverRequest{
			LatitudeOrigin:  -6.200000,
			LongitudeOrigin: 106.816666,
			Radius:          5,
		}

		mockRepo.EXPECT().SearchNearbyDrivers(ctx, req).Return([]redis.GeoLocation{}, nil)

		drivers, err := svc.FindNearbyDrivers(ctx, req)
		assert.NoError(t, err)
		assert.Empty(t, drivers)
	})

	t.Run("Error - Radius invalid", func(t *testing.T) {
		req := model.SearchDriverRequest{
			LatitudeOrigin:  -6.200000,
			LongitudeOrigin: 106.816666,
			Radius:          0, // invalid
		}

		drivers, err := svc.FindNearbyDrivers(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "invalid radius", err.Error())
		assert.Nil(t, drivers)
	})

	t.Run("Error - Repository gagal", func(t *testing.T) {
		req := model.SearchDriverRequest{
			LatitudeOrigin:  -6.2,
			LongitudeOrigin: 106.8,
			Radius:          5,
		}

		mockRepo.EXPECT().
			SearchNearbyDrivers(ctx, req).
			Return(nil, errors.New("redis error"))

		drivers, err := svc.FindNearbyDrivers(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, drivers)
	})
}

func TestGetDriverLocation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockLocationRepository(ctrl)
	mockDriverClient := mock.NewMockDriverServiceClient(ctrl)
	svc := service.NewLocationService(mockRepo, mockDriverClient)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		ts := time.Now()
		mockResponse := &model.TrackLocationResponse{
			DriverID:  "driver1",
			Latitude:  -6.200000,
			Longitude: 106.816666,
			Timestamp: ts,
		}

		mockRepo.EXPECT().TrackDriver(ctx, "driver1").Return(mockResponse, nil)

		res, err := svc.GetDriverLocation(ctx, "driver1", "order1")
		assert.NoError(t, err)
		assert.Equal(t, mockResponse, res)
	})

	t.Run("Error - Driver tidak ditemukan", func(t *testing.T) {
		mockRepo.EXPECT().TrackDriver(ctx, "driver999").Return(nil, errors.New("driver location not found"))

		res, err := svc.GetDriverLocation(ctx, "driver999", "order1")
		assert.Error(t, err)
		assert.Equal(t, "driver location not found", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - DriverID kosong", func(t *testing.T) {
		res, err := svc.GetDriverLocation(ctx, "", "order1")

		assert.Error(t, err)
		assert.Equal(t, "driver_id is required", err.Error())
		assert.Nil(t, res)
	})
}

func TestGetEmergencyLocation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockLocationRepository(ctrl)
	mockDriverClient := mock.NewMockDriverServiceClient(ctrl)
	svc := service.NewLocationService(mockRepo, mockDriverClient)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		ts := time.Now()
		mockResponse := &model.TrackLocationResponse{
			DriverID:  "driver-emergency",
			Latitude:  -6.175392,
			Longitude: 106.827153,
			Timestamp: ts,
		}

		mockRepo.EXPECT().TrackDriver(ctx, "driver-emergency").Return(mockResponse, nil)

		res, err := svc.GetEmergencyLocation(ctx, "driver-emergency", "order1", "passenger")
		assert.NoError(t, err)
		assert.Equal(t, mockResponse, res)
	})

	t.Run("Error - Driver tidak ditemukan", func(t *testing.T) {
		mockRepo.EXPECT().TrackDriver(ctx, "driver-not-found").Return(nil, errors.New("driver location not found"))

		res, err := svc.GetEmergencyLocation(ctx, "driver-not-found", "order1", "passenger")
		assert.Error(t, err)
		assert.Equal(t, "driver location not found", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - DriverID kosong", func(t *testing.T) {
		res, err := svc.GetEmergencyLocation(ctx, "", "order1", "user")

		assert.Error(t, err)
		assert.Equal(t, "driver_id is required", err.Error())
		assert.Nil(t, res)
	})
}
