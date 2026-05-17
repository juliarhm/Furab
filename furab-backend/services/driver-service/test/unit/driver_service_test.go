package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/driver-service/internal/model"
	"furab-backend/services/driver-service/internal/repository"
	"furab-backend/services/driver-service/internal/service"
)

type MockDriverRepository struct {
	SaveFunc           func(ctx context.Context, driver *model.Driver) error
	FindByIDFunc       func(ctx context.Context, driverID string) (*model.Driver, error)
	UpdateFunc         func(ctx context.Context, driver *model.Driver) error
	UpdateStatusFunc   func(ctx context.Context, driverID, status string) error
	UpdateLocationFunc func(ctx context.Context, driverID string, lat, long float64) error
}

func (m *MockDriverRepository) Save(ctx context.Context, driver *model.Driver) error {
	return m.SaveFunc(ctx, driver)
}
func (m *MockDriverRepository) FindByID(ctx context.Context, driverID string) (*model.Driver, error) {
	return m.FindByIDFunc(ctx, driverID)
}
func (m *MockDriverRepository) Update(ctx context.Context, driver *model.Driver) error {
	return m.UpdateFunc(ctx, driver)
}
func (m *MockDriverRepository) UpdateStatus(ctx context.Context, driverID, status string) error {
	return m.UpdateStatusFunc(ctx, driverID, status)
}
func (m *MockDriverRepository) UpdateLocation(ctx context.Context, driverID string, lat, long float64) error {
	return m.UpdateLocationFunc(ctx, driverID, lat, long)
}

func TestDriverService_CreateDriver(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &MockDriverRepository{
			SaveFunc: func(ctx context.Context, driver *model.Driver) error { return nil },
		}
		svc := service.NewDriverService(repo)

		req := &model.CreateDriverRequest{
			DriverID:    "1",
			Name:        "Driver1",
			Phone:       "08123",
			VehicleType: "Car",
		}
		res, err := svc.CreateDriver(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.DriverID != "1" {
			t.Errorf("Expected driver id 1")
		}
	})

	t.Run("Error Validation", func(t *testing.T) {
		repo := &MockDriverRepository{}
		svc := service.NewDriverService(repo)

		req := &model.CreateDriverRequest{DriverID: ""}
		_, err := svc.CreateDriver(context.Background(), req)
		if err == nil {
			t.Fatalf("Expected error")
		}
	})
}

func TestDriverService_GetDriver(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &MockDriverRepository{
			FindByIDFunc: func(ctx context.Context, driverID string) (*model.Driver, error) {
				return &model.Driver{DriverID: "1"}, nil
			},
		}
		svc := service.NewDriverService(repo)

		res, err := svc.GetDriver(context.Background(), "1")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.DriverID != "1" {
			t.Errorf("Expected driver id 1")
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		repo := &MockDriverRepository{
			FindByIDFunc: func(ctx context.Context, driverID string) (*model.Driver, error) {
				return nil, repository.ErrDriverNotFound
			},
		}
		svc := service.NewDriverService(repo)

		_, err := svc.GetDriver(context.Background(), "99")
		if err == nil || !errors.Is(err, service.ErrDriverNotFound) {
			t.Fatalf("Expected not found error")
		}
	})
}

func TestDriverService_UpdateDriver(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &MockDriverRepository{
			FindByIDFunc: func(ctx context.Context, driverID string) (*model.Driver, error) {
				return &model.Driver{DriverID: "1"}, nil
			},
			UpdateFunc: func(ctx context.Context, driver *model.Driver) error { return nil },
		}
		svc := service.NewDriverService(repo)

		req := &model.UpdateDriverRequest{Name: "New", Phone: "08123", VehicleType: "Car"}
		res, err := svc.UpdateDriver(context.Background(), "1", req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.Status != "success" {
			t.Errorf("Expected success")
		}
	})
}

func TestDriverService_UpdateStatus(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &MockDriverRepository{
			FindByIDFunc: func(ctx context.Context, driverID string) (*model.Driver, error) {
				return &model.Driver{DriverID: "1"}, nil
			},
			UpdateStatusFunc: func(ctx context.Context, driverID, status string) error { return nil },
		}
		svc := service.NewDriverService(repo)

		res, err := svc.UpdateStatus(context.Background(), "1", "ONLINE")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.Status != "success" {
			t.Errorf("Expected success")
		}
	})
}

func TestDriverService_UpdateLocation(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		repo := &MockDriverRepository{
			FindByIDFunc: func(ctx context.Context, driverID string) (*model.Driver, error) {
				return &model.Driver{DriverID: "1"}, nil
			},
			UpdateLocationFunc: func(ctx context.Context, driverID string, lat, long float64) error { return nil },
		}
		svc := service.NewDriverService(repo)

		res, err := svc.UpdateLocation(context.Background(), "1", 10.0, 20.0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.Status != "success" {
			t.Errorf("Expected success")
		}
	})
}
