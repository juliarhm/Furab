package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/notification-service/internal/model"
	"furab-backend/services/notification-service/internal/service"
	"furab-backend/services/notification-service/test/unit/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newTestService helper sets up the service and its dependencies
func newTestService(t *testing.T) (
	service.NotificationService,
	*mock.MockNotificationRepository,
	*mock.MockEmailClient,
	*gomock.Controller,
) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockNotificationRepository(ctrl)
	mockEmailClient := mock.NewMockEmailClient(ctrl)

	svc := service.NewNotificationService(mockRepo, mockEmailClient)

	return svc, mockRepo, mockEmailClient, ctrl
}

func TestSendNotification(t *testing.T) {
	t.Run("Success - Push Notification", func(t *testing.T) {
		svc, mockRepo, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "chat.message",
			ReceiverID:  "user1",
			Channel:     "push",
			ReferenceID: "msg-1",
		}

		template := &model.NotifTemplate{
			EventType:       "chat.message",
			TitleTemplate:   "New Message",
			MessageTemplate: "You have a new message",
		}

		mockRepo.EXPECT().GetTemplateByEventType(ctx, "chat.message").Return(template, nil)
		
		// Verify log content is correct
		mockRepo.EXPECT().SaveNotificationLog(ctx, gomock.AssignableToTypeOf(model.NotificationLog{})).DoAndReturn(func(ctx context.Context, log model.NotificationLog) error {
			assert.Equal(t, "user1", log.ReceiverID)
			assert.Equal(t, "push", log.Channel)
			assert.Equal(t, "New Message", log.Title)
			assert.Equal(t, "You have a new message", log.Message)
			assert.Equal(t, "msg-1", log.ReferenceID)
			assert.Equal(t, "sent", log.Status)
			assert.NotEmpty(t, log.NotificationID)
			assert.False(t, log.Timestamp.IsZero())
			return nil
		})

		res, err := svc.SendNotification(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "success", res.Status)
		assert.NotEmpty(t, res.NotificationID)
		assert.False(t, res.Timestamp.IsZero())
		assert.Equal(t, "push", res.Channel)
		assert.Equal(t, "user1", res.ReceiverID)
	})

	t.Run("Success - Email Notification", func(t *testing.T) {
		svc, mockRepo, mockEmail, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "ride.created",
			ReceiverID:  "user2",
			Channel:     "email",
			ReferenceID: "ride-1",
		}

		template := &model.NotifTemplate{
			EventType:       "ride.created",
			TitleTemplate:   "Ride Created",
			MessageTemplate: "Your ride has been created",
		}

		mockRepo.EXPECT().GetTemplateByEventType(ctx, "ride.created").Return(template, nil)
		
		mockRepo.EXPECT().SaveNotificationLog(ctx, gomock.AssignableToTypeOf(model.NotificationLog{})).Return(nil)
		mockEmail.EXPECT().SendEmail(ctx, "user2", template.TitleTemplate, template.MessageTemplate).Return(nil)

		res, err := svc.SendNotification(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "success", res.Status)
		assert.NotEmpty(t, res.NotificationID)
		assert.False(t, res.Timestamp.IsZero())
		assert.Equal(t, "email", res.Channel)
		assert.Equal(t, "user2", res.ReceiverID)
	})

	t.Run("Error - receiver_id kosong", func(t *testing.T) {
		svc, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()
		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "chat.message",
			ReceiverID:  "", // empty
			Channel:     "push",
			ReferenceID: "msg-1",
		}

		res, err := svc.SendNotification(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "receiver_id is required", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - reference_id kosong", func(t *testing.T) {
		svc, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()
		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "chat.message",
			ReceiverID:  "user1",
			Channel:     "push",
			ReferenceID: "", // empty
		}

		res, err := svc.SendNotification(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "reference_id is required", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - channel kosong", func(t *testing.T) {
		svc, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()
		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "chat.message",
			ReceiverID:  "user1",
			Channel:     "", // empty
			ReferenceID: "msg-1",
		}

		res, err := svc.SendNotification(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "channel is required", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Invalid Request (event_type kosong)", func(t *testing.T) {
		svc, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "", // empty event_type
			ReceiverID:  "user1",
			Channel:     "push",
			ReferenceID: "msg-1",
		}

		res, err := svc.SendNotification(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "event type is required", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Channel tidak valid", func(t *testing.T) {
		svc, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "chat.message",
			ReceiverID:  "user1",
			Channel:     "sms", // invalid channel
			ReferenceID: "msg-1",
		}

		res, err := svc.SendNotification(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "invalid channel", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Template nil", func(t *testing.T) {
		svc, mockRepo, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "chat.message",
			ReceiverID:  "user1",
			Channel:     "push",
			ReferenceID: "msg-1",
		}

		mockRepo.EXPECT().GetTemplateByEventType(ctx, "chat.message").Return(nil, nil)

		res, err := svc.SendNotification(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "template not found", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Unknown event type", func(t *testing.T) {
		svc, mockRepo, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "unknown.event",
			ReceiverID:  "user1",
			Channel:     "push",
			ReferenceID: "msg-1",
		}

		mockRepo.EXPECT().GetTemplateByEventType(ctx, "unknown.event").Return(nil, nil)

		res, err := svc.SendNotification(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "template not found", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Get Template gagal db error", func(t *testing.T) {
		svc, mockRepo, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "chat.message",
			ReceiverID:  "user1",
			Channel:     "push",
			ReferenceID: "msg-1",
		}

		mockRepo.EXPECT().GetTemplateByEventType(ctx, "chat.message").Return(nil, errors.New("template not found error"))

		res, err := svc.SendNotification(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "template not found error", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Repository gagal (SaveLog)", func(t *testing.T) {
		svc, mockRepo, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "chat.message",
			ReceiverID:  "user1",
			Channel:     "push",
			ReferenceID: "msg-1",
		}

		template := &model.NotifTemplate{
			EventType:       "chat.message",
			TitleTemplate:   "New Message",
			MessageTemplate: "You have a new message",
		}

		mockRepo.EXPECT().GetTemplateByEventType(ctx, "chat.message").Return(template, nil)
		mockRepo.EXPECT().SaveNotificationLog(ctx, gomock.Any()).Return(errors.New("db save error"))

		res, err := svc.SendNotification(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "db save error", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Email gagal", func(t *testing.T) {
		svc, mockRepo, mockEmail, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.EventNotificationRequest{
			EventType:   "ride.created",
			ReceiverID:  "user2",
			Channel:     "email",
			ReferenceID: "ride-1",
		}

		template := &model.NotifTemplate{
			EventType:       "ride.created",
			TitleTemplate:   "Ride Created",
			MessageTemplate: "Your ride has been created",
		}

		mockRepo.EXPECT().GetTemplateByEventType(ctx, "ride.created").Return(template, nil)
		
		// First call when saving initial "sent" status
		mockRepo.EXPECT().SaveNotificationLog(ctx, gomock.Any()).Return(nil)
		
		// Expect Email error
		mockEmail.EXPECT().SendEmail(ctx, "user2", template.TitleTemplate, template.MessageTemplate).Return(errors.New("email error"))
		
		// Expect second call to update status to "failed"
		mockRepo.EXPECT().SaveNotificationLog(ctx, gomock.AssignableToTypeOf(model.NotificationLog{})).DoAndReturn(func(ctx context.Context, log model.NotificationLog) error {
			assert.Equal(t, "failed", log.Status)
			return nil
		})

		res, err := svc.SendNotification(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "email error", err.Error())
		assert.Nil(t, res)
	})
}
