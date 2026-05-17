package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/chat-service/internal/model"
	"furab-backend/services/chat-service/internal/service"
	"furab-backend/services/chat-service/test/unit/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newTestService helper sets up the service and its dependencies
func newTestService(t *testing.T) (
	service.ChatService,
	*mock.MockChatRepository,
	*mock.MockUserServiceClient,
	*mock.MockDriverServiceClient,
	*mock.MockNotificationClient,
	*gomock.Controller,
) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockChatRepository(ctrl)
	mockUserClient := mock.NewMockUserServiceClient(ctrl)
	mockDriverClient := mock.NewMockDriverServiceClient(ctrl)
	mockNotifClient := mock.NewMockNotificationClient(ctrl)

	svc := service.NewChatService(mockRepo, mockUserClient, mockDriverClient, mockNotifClient)

	return svc, mockRepo, mockUserClient, mockDriverClient, mockNotifClient, ctrl
}

func TestSendMessage(t *testing.T) {
	t.Run("Success (user -> driver)", func(t *testing.T) {
		svc, mockRepo, mockUser, _, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.SendMessageRequest{
			OrderID:     "order1",
			SenderID:    "user1",
			SenderType:  "user",
			ReceiverID:  "driver1",
			MessageText: "halo",
		}

		mockUser.EXPECT().ValidateUser(ctx, "user1").Return(true, nil)
		mockRepo.EXPECT().SaveMessage(ctx, gomock.Any()).Return(nil)
		mockNotif.EXPECT().SendNotification(ctx, "driver1", "halo").Return(nil)

		res, err := svc.SendMessage(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "user1", res.SenderID)
		assert.Equal(t, "halo", res.MessageText)
		assert.Equal(t, "sent", res.Status)
		assert.NotEmpty(t, res.MessageID)
	})

	t.Run("Success (driver -> user)", func(t *testing.T) {
		svc, mockRepo, _, mockDriver, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.SendMessageRequest{
			OrderID:     "order1",
			SenderID:    "driver1",
			SenderType:  "driver",
			ReceiverID:  "user1",
			MessageText: "otw",
		}

		mockDriver.EXPECT().ValidateDriver(ctx, "driver1").Return(true, nil)
		mockRepo.EXPECT().SaveMessage(ctx, gomock.Any()).Return(nil)
		mockNotif.EXPECT().SendNotification(ctx, "user1", "otw").Return(nil)

		res, err := svc.SendMessage(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "driver1", res.SenderID)
		assert.Equal(t, "otw", res.MessageText)
	})

	t.Run("Error: field kosong", func(t *testing.T) {
		svc, _, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.SendMessageRequest{
			OrderID: "", // missing order ID
		}

		res, err := svc.SendMessage(ctx, req)
		assert.ErrorIs(t, err, service.ErrInvalidRequest)
		assert.Nil(t, res)
	})

	t.Run("Error: sender tidak valid", func(t *testing.T) {
		svc, _, mockUser, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.SendMessageRequest{
			OrderID:     "order1",
			SenderID:    "user_invalid",
			SenderType:  "user",
			ReceiverID:  "driver1",
			MessageText: "halo",
		}

		mockUser.EXPECT().ValidateUser(ctx, "user_invalid").Return(false, nil)

		res, err := svc.SendMessage(ctx, req)
		assert.ErrorIs(t, err, service.ErrInvalidSender)
		assert.Nil(t, res)
	})

	t.Run("Error: repo gagal", func(t *testing.T) {
		svc, mockRepo, mockUser, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.SendMessageRequest{
			OrderID:     "order1",
			SenderID:    "user1",
			SenderType:  "user",
			ReceiverID:  "driver1",
			MessageText: "halo",
		}

		mockUser.EXPECT().ValidateUser(ctx, "user1").Return(true, nil)
		mockRepo.EXPECT().SaveMessage(ctx, gomock.Any()).Return(errors.New("db error"))

		res, err := svc.SendMessage(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error: notification gagal", func(t *testing.T) {
		svc, mockRepo, mockUser, _, mockNotif, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.SendMessageRequest{
			OrderID:     "order1",
			SenderID:    "user1",
			SenderType:  "user",
			ReceiverID:  "driver1",
			MessageText: "halo",
		}

		mockUser.EXPECT().ValidateUser(ctx, "user1").Return(true, nil)
		mockRepo.EXPECT().SaveMessage(ctx, gomock.Any()).Return(nil)
		mockNotif.EXPECT().SendNotification(ctx, "driver1", "halo").Return(errors.New("notif failed"))

		res, err := svc.SendMessage(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "notif failed", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error: sender_type tidak valid", func(t *testing.T) {
		svc, _, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		req := model.SendMessageRequest{
			OrderID:     "order1",
			SenderID:    "user1",
			SenderType:  "admin",
			ReceiverID:  "driver1",
			MessageText: "halo",
		}

		res, err := svc.SendMessage(ctx, req)

		assert.ErrorIs(t, err, service.ErrInvalidRequest)
		assert.Nil(t, res)
	})
}

func TestUpdateMessageStatus(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		svc, mockRepo, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.ReadReceiptRequest{
			MessageID: "msg1",
			OrderID:   "order1",
			Status:    "read",
		}

		mockRepo.EXPECT().UpdateMessageStatus(ctx, "msg1", "read").Return(nil)

		err := svc.UpdateMessageStatus(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("Error: message_id kosong", func(t *testing.T) {
		svc, _, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.ReadReceiptRequest{
			MessageID: "",
			OrderID:   "order1",
			Status:    "read",
		}

		err := svc.UpdateMessageStatus(ctx, req)
		assert.ErrorIs(t, err, service.ErrInvalidRequest)
	})

	t.Run("Error: repo gagal", func(t *testing.T) {
		svc, mockRepo, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.ReadReceiptRequest{
			MessageID: "msg1",
			OrderID:   "order1",
			Status:    "delivered",
		}

		mockRepo.EXPECT().UpdateMessageStatus(ctx, "msg1", "delivered").Return(errors.New("db error"))

		err := svc.UpdateMessageStatus(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})
}

func TestGetChatHistory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		svc, mockRepo, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		messages := []model.Message{
			{MessageID: "msg1", Content: "halo"},
			{MessageID: "msg2", Content: "iya"},
		}

		mockRepo.EXPECT().GetMessagesByOrderID(ctx, "order1").Return(messages, nil)

		res, err := svc.GetChatHistory(ctx, "order1")
		assert.NoError(t, err)
		assert.Len(t, res, 2)
	})

	t.Run("Success: empty chat", func(t *testing.T) {
		svc, mockRepo, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockRepo.EXPECT().GetMessagesByOrderID(ctx, "order1").Return([]model.Message{}, nil)

		res, err := svc.GetChatHistory(ctx, "order1")
		assert.NoError(t, err)
		assert.Empty(t, res)
	})

	t.Run("Error: repo gagal", func(t *testing.T) {
		svc, mockRepo, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockRepo.EXPECT().GetMessagesByOrderID(ctx, "order1").Return(nil, errors.New("db error"))

		res, err := svc.GetChatHistory(ctx, "order1")
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Nil(t, res)
	})
}

func TestCloseChatSession(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		svc, mockRepo, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockRepo.EXPECT().CloseSession(ctx, "order1").Return(nil)

		err := svc.CloseChatSession(ctx, "order1")
		assert.NoError(t, err)
	})

	t.Run("Error: order_id kosong", func(t *testing.T) {
		svc, _, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		err := svc.CloseChatSession(ctx, "")
		assert.ErrorIs(t, err, service.ErrInvalidRequest)
	})

	t.Run("Error: repo gagal", func(t *testing.T) {
		svc, mockRepo, _, _, _, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockRepo.EXPECT().CloseSession(ctx, "order1").Return(errors.New("db error"))

		err := svc.CloseChatSession(ctx, "order1")
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})
}
