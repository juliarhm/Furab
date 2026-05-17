package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"furab-backend/services/emergency-service/internal/model"
	"furab-backend/services/emergency-service/internal/service"
	"furab-backend/services/emergency-service/test/unit/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (
	service.EmergencyService,
	*mock.MockEmergencyRepository,
	*mock.MockLocationClient,
	*mock.MockActorClient,
	*mock.MockNotificationClient,
	*gomock.Controller,
) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockEmergencyRepository(ctrl)
	mockLocationClient := mock.NewMockLocationClient(ctrl)
	mockActorClient := mock.NewMockActorClient(ctrl)
	mockNotificationClient := mock.NewMockNotificationClient(ctrl)

	svc := service.NewEmergencyServiceWithDependencies(mockRepo, mockLocationClient, mockActorClient, mockNotificationClient)

	return svc, mockRepo, mockLocationClient, mockActorClient, mockNotificationClient, ctrl
}

func TestTriggerEmergency(t *testing.T) {
	t.Run("Success - User Trigger Emergency", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:       "user1",
			ActorType:     "user",
			OrderID:       "order1",
			Latitude:      -6.200000,
			Longitude:     106.816666,
			EmergencyType: "accident",
			Timestamp:     time.Now(),
		}

		location := &model.EmergencyLocation{
			Latitude:  -6.200000,
			Longitude: 106.816666,
		}

		contact := &model.EmergencyContact{
			ReceiverID: "contact1",
			Phone:      "08123456789",
		}

		gomock.InOrder(
			mockActor.EXPECT().ValidateActor(ctx, "user1", "user").Return(true, nil),
			mockActor.EXPECT().ValidateOrder(ctx, "order1").Return(true, nil),
			mockLoc.EXPECT().GetLastLocation(ctx, "user1", "user").Return(location, nil),
			mockRepo.EXPECT().SaveEmergencyEvent(ctx, gomock.Any()).Return(nil),
			mockActor.EXPECT().GetEmergencyContact(ctx, "user1", "user").Return(contact, nil),
			mockNotif.EXPECT().SendNotification(ctx, gomock.AssignableToTypeOf(model.EmergencyNotification{})).DoAndReturn(func(ctx context.Context, n model.EmergencyNotification) error {
				assert.Equal(t, "high", n.Priority)
				assert.Contains(t, n.LocationURL, "-6.200000")
				assert.Contains(t, n.LocationURL, "106.816666")
				return nil
			}),
			mockNotif.EXPECT().SendEmergencyContact(ctx, *contact, gomock.Any()).Return(nil),
		)

		res, err := svc.TriggerEmergency(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.EmergencyID)
		assert.Equal(t, "success", res.Status)
		assert.Equal(t, "emergency created", res.Message)
	})

	t.Run("Success - Driver Trigger Emergency", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:       "driver1",
			ActorType:     "driver",
			OrderID:       "order1",
			Latitude:      -6.200000,
			Longitude:     106.816666,
			EmergencyType: "accident",
			Timestamp:     time.Now(),
		}

		location := &model.EmergencyLocation{
			Latitude:  -6.200000,
			Longitude: 106.816666,
		}

		gomock.InOrder(
			mockActor.EXPECT().ValidateActor(ctx, "driver1", "driver").Return(true, nil),
			mockActor.EXPECT().ValidateOrder(ctx, "order1").Return(true, nil),
			mockLoc.EXPECT().GetLastLocation(ctx, "driver1", "driver").Return(location, nil),
			mockRepo.EXPECT().SaveEmergencyEvent(ctx, gomock.Any()).Return(nil),
			mockActor.EXPECT().GetEmergencyContact(ctx, "driver1", "driver").Return(nil, nil), // No contact
			mockNotif.EXPECT().SendNotification(ctx, gomock.AssignableToTypeOf(model.EmergencyNotification{})).Return(nil),
		)

		res, err := svc.TriggerEmergency(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.EmergencyID)
		assert.Equal(t, "success", res.Status)
		assert.Equal(t, "emergency created", res.Message)
	})

	t.Run("Error - Field kosong", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID: "", // kosong
		}

		mockActor.EXPECT().ValidateActor(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		mockActor.EXPECT().ValidateOrder(gomock.Any(), gomock.Any()).Times(0)
		mockLoc.EXPECT().GetLastLocation(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		mockRepo.EXPECT().SaveEmergencyEvent(gomock.Any(), gomock.Any()).Times(0)
		mockNotif.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.TriggerEmergency(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "invalid actor", err.Error())
		assert.NotNil(t, res) // response is returned as per code
		assert.Equal(t, "failed", res.Status)
	})

	t.Run("Error - Actor type tidak valid", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:   "user1",
			ActorType: "admin", // tidak valid
		}

		mockActor.EXPECT().ValidateActor(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
		mockRepo.EXPECT().SaveEmergencyEvent(gomock.Any(), gomock.Any()).Times(0)
		mockNotif.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Times(0)
		mockLoc.EXPECT().GetLastLocation(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.TriggerEmergency(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "invalid actor", err.Error())
		assert.Equal(t, "failed", res.Status)
	})

	t.Run("Error - User tidak valid", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:   "user1",
			ActorType: "user",
		}

		mockActor.EXPECT().ValidateActor(ctx, "user1", "user").Return(false, nil)
		mockRepo.EXPECT().SaveEmergencyEvent(gomock.Any(), gomock.Any()).Times(0)
		mockNotif.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Times(0)
		mockLoc.EXPECT().GetLastLocation(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.TriggerEmergency(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "invalid actor", err.Error())
		assert.Equal(t, "failed", res.Status)
	})

	t.Run("Error - Order tidak valid", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:   "user1",
			ActorType: "user",
			OrderID:   "order-invalid",
		}

		// As per emergency_service.go, Order validation is best-effort.
		// If it fails, flow continues.
		mockActor.EXPECT().ValidateActor(ctx, "user1", "user").Return(true, nil)
		mockActor.EXPECT().ValidateOrder(ctx, "order-invalid").Return(false, errors.New("invalid order"))
		
		mockLoc.EXPECT().GetLastLocation(ctx, "user1", "user").Return(&model.EmergencyLocation{}, nil)
		mockRepo.EXPECT().SaveEmergencyEvent(ctx, gomock.Any()).Return(nil)
		mockActor.EXPECT().GetEmergencyContact(ctx, "user1", "user").Return(nil, nil)
		mockNotif.EXPECT().SendNotification(ctx, gomock.Any()).Return(nil)

		res, err := svc.TriggerEmergency(ctx, req)

		assert.NoError(t, err) // It continues despite order invalid
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.EmergencyID)
		assert.Equal(t, "success", res.Status)
		assert.Equal(t, "emergency created", res.Message)
	})

	t.Run("Error - Repository gagal", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:   "user1",
			ActorType: "user",
		}

		mockActor.EXPECT().ValidateActor(ctx, "user1", "user").Return(true, nil)
		mockLoc.EXPECT().GetLastLocation(ctx, "user1", "user").Return(&model.EmergencyLocation{}, nil)
		mockRepo.EXPECT().SaveEmergencyEvent(ctx, gomock.Any()).Return(errors.New("db error"))
		
		// Ensure notification is not called
		mockNotif.EXPECT().SendNotification(gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.TriggerEmergency(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Notification gagal", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:   "user1",
			ActorType: "user",
		}

		mockActor.EXPECT().ValidateActor(ctx, "user1", "user").Return(true, nil)
		mockLoc.EXPECT().GetLastLocation(ctx, "user1", "user").Return(&model.EmergencyLocation{}, nil)
		
		// Emergency is saved successfully
		mockRepo.EXPECT().SaveEmergencyEvent(ctx, gomock.AssignableToTypeOf(model.EmergencyEvent{})).DoAndReturn(func(ctx context.Context, ev model.EmergencyEvent) error {
			assert.Equal(t, "active", ev.Status)
			return nil
		})

		mockActor.EXPECT().GetEmergencyContact(ctx, "user1", "user").Return(nil, nil)
		
		// Notification fails, but the service ignores the error and continues
		mockNotif.EXPECT().SendNotification(ctx, gomock.Any()).Return(errors.New("notif error"))

		res, err := svc.TriggerEmergency(ctx, req)

		// The service design is best-effort, so it will actually return success without error
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.EmergencyID)
		assert.Equal(t, "success", res.Status)
		assert.Equal(t, "emergency created", res.Message)
	})

	t.Run("Error - Emergency Contact gagal", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:   "user1",
			ActorType: "user",
		}

		contact := &model.EmergencyContact{ReceiverID: "c1"}

		mockActor.EXPECT().ValidateActor(ctx, "user1", "user").Return(true, nil)
		mockLoc.EXPECT().GetLastLocation(ctx, "user1", "user").Return(&model.EmergencyLocation{}, nil)
		mockRepo.EXPECT().SaveEmergencyEvent(ctx, gomock.Any()).Return(nil)
		
		mockActor.EXPECT().GetEmergencyContact(ctx, "user1", "user").Return(contact, nil)
		
		// Notification continues
		mockNotif.EXPECT().SendNotification(ctx, gomock.Any()).Return(nil)
		// Emergency contact fails
		mockNotif.EXPECT().SendEmergencyContact(ctx, *contact, gomock.Any()).Return(errors.New("contact error"))

		res, err := svc.TriggerEmergency(ctx, req)

		// Again, the service ignores this error, so it should succeed
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.EmergencyID)
		assert.Equal(t, "success", res.Status)
		assert.Equal(t, "emergency created", res.Message)
	})

	t.Run("Success - Tanpa emergency_type", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:   "user1",
			ActorType: "user",
			// No EmergencyType
		}

		mockActor.EXPECT().ValidateActor(ctx, "user1", "user").Return(true, nil)
		mockLoc.EXPECT().GetLastLocation(ctx, "user1", "user").Return(&model.EmergencyLocation{}, nil)
		
		mockRepo.EXPECT().SaveEmergencyEvent(ctx, gomock.AssignableToTypeOf(model.EmergencyEvent{})).DoAndReturn(func(ctx context.Context, ev model.EmergencyEvent) error {
			assert.Equal(t, "", ev.EmergencyType) // Ensure it's empty but successful
			return nil
		})
		
		mockActor.EXPECT().GetEmergencyContact(ctx, "user1", "user").Return(nil, nil)
		mockNotif.EXPECT().SendNotification(ctx, gomock.Any()).Return(nil)

		res, err := svc.TriggerEmergency(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.EmergencyID)
		assert.Equal(t, "success", res.Status)
		assert.Equal(t, "emergency created", res.Message)
	})

	t.Run("Success - Fallback ke request location", func(t *testing.T) {
		svc, mockRepo, mockLoc, mockActor, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.TriggerEmergencyRequest{
			ActorID:   "user1",
			ActorType: "user",
			Latitude:  -6.2,
			Longitude: 106.8,
		}

		mockActor.EXPECT().ValidateActor(ctx, "user1", "user").Return(true, nil)

		mockLoc.EXPECT().GetLastLocation(ctx, "user1", "user").
			Return(nil, errors.New("location down"))

		mockRepo.EXPECT().SaveEmergencyEvent(ctx, gomock.AssignableToTypeOf(model.EmergencyEvent{})).
			DoAndReturn(func(ctx context.Context, ev model.EmergencyEvent) error {
				assert.Equal(t, -6.2, ev.Latitude)
				assert.Equal(t, 106.8, ev.Longitude)
				return nil
			})

		mockActor.EXPECT().GetEmergencyContact(ctx, "user1", "user").Return(nil, nil)
		mockNotif.EXPECT().SendNotification(ctx, gomock.Any()).Return(nil)

		res, err := svc.TriggerEmergency(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.EmergencyID)
		assert.Equal(t, "success", res.Status)
		assert.Equal(t, "emergency created", res.Message)
	})
}
