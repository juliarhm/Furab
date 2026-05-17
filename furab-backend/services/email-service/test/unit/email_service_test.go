package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/email-service/internal/model"
	"furab-backend/services/email-service/internal/service"
	"furab-backend/services/email-service/test/unit/mock"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (
	service.EmailService,
	*mock.MockEmailRepository,
	*mock.MockEmailSender,
	*gomock.Controller,
) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockEmailRepository(ctrl)
	mockEmailSender := mock.NewMockEmailSender(ctrl)

	svc := service.NewEmailService(mockRepo, mockEmailSender)

	return svc, mockRepo, mockEmailSender, ctrl
}

func TestSendEmail(t *testing.T) {
	t.Run("Success - Send Email", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			Subject:       "Welcome",
			Body:          "Hello user!",
			ReceiverID:    "user1",
			ReferenceID:   "ref-1",
		}

		gomock.InOrder(
			mockRepo.EXPECT().SaveEmailLog(ctx, gomock.AssignableToTypeOf(model.EmailLog{})).DoAndReturn(func(ctx context.Context, log model.EmailLog) error {
				assert.Equal(t, "sent", log.Status)
				return nil
			}),
			mockClient.EXPECT().Send(ctx, "user@example.com", "Welcome", "Hello user!").Return(nil),
		)

		res, err := svc.SendEmail(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, req.ReceiverEmail, res.ReceiverEmail)
		assert.Equal(t, req.Subject, res.Subject)
		assert.Equal(t, "sent", res.Status)
		assert.NotEmpty(t, res.EmailID)
		assert.False(t, res.Timestamp.IsZero())
	})

	t.Run("Error - Invalid Request (receiver_email kosong)", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "",
			Subject:       "Welcome",
			Body:          "Hello user!",
		}

		mockRepo.EXPECT().SaveEmailLog(gomock.Any(), gomock.Any()).Times(0)
		mockClient.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "invalid request", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Invalid Request (subject kosong)", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			Subject:       "",
			Body:          "Hello user!",
		}

		mockRepo.EXPECT().SaveEmailLog(gomock.Any(), gomock.Any()).Times(0)
		mockClient.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "invalid request", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Invalid Request (body kosong tapi subject ada)", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			Subject:       "Welcome",
			Body:          "",
		}

		mockRepo.EXPECT().SaveEmailLog(gomock.Any(), gomock.Any()).Times(0)
		mockClient.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "invalid request", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Repository gagal", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			Subject:       "Welcome",
			Body:          "Hello user!",
		}

		mockRepo.EXPECT().SaveEmailLog(ctx, gomock.Any()).Return(errors.New("db error"))
		
		mockClient.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Email client gagal", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			Subject:       "Welcome",
			Body:          "Hello user!",
		}

		gomock.InOrder(
			mockRepo.EXPECT().SaveEmailLog(ctx, gomock.AssignableToTypeOf(model.EmailLog{})).DoAndReturn(func(ctx context.Context, log model.EmailLog) error {
				assert.Equal(t, "sent", log.Status)
				return nil
			}),
			mockClient.EXPECT().Send(ctx, "user@example.com", "Welcome", "Hello user!").Return(errors.New("smtp timeout")),
			mockRepo.EXPECT().SaveEmailLog(ctx, gomock.AssignableToTypeOf(model.EmailLog{})).DoAndReturn(func(ctx context.Context, log model.EmailLog) error {
				assert.Equal(t, "failed", log.Status)
				return nil
			}),
		)

		res, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "smtp timeout", err.Error())
		assert.Nil(t, res)
	})

	// New tests for TemplateID and Metadata
	t.Run("Success - Dengan TemplateID", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			TemplateID:    "tpl-123",
		}

		template := &model.EmailTemplate{
			TemplateID: "tpl-123",
			Subject:    "Template Subject",
			Body:       "Template Body",
		}

		gomock.InOrder(
			mockRepo.EXPECT().GetTemplateByID(ctx, "tpl-123").Return(template, nil),
			mockRepo.EXPECT().SaveEmailLog(ctx, gomock.AssignableToTypeOf(model.EmailLog{})).DoAndReturn(func(ctx context.Context, log model.EmailLog) error {
				assert.Equal(t, "Template Subject", log.Subject)
				return nil
			}),
			mockClient.EXPECT().Send(ctx, "user@example.com", "Template Subject", "Template Body").Return(nil),
		)

		res, err := svc.SendEmail(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "Template Subject", res.Subject)
	})

	t.Run("Error - Template tidak ditemukan", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			TemplateID:    "tpl-unknown",
		}

		mockRepo.EXPECT().GetTemplateByID(ctx, "tpl-unknown").Return(nil, errors.New("not found"))
		mockRepo.EXPECT().SaveEmailLog(gomock.Any(), gomock.Any()).Times(0)
		mockClient.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.SendEmail(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "not found", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Success - Metadata", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			Subject:       "Welcome",
			Body:          "Hello user!",
			Metadata: map[string]interface{}{
				"source": "api",
				"priority": 1,
			},
		}

		gomock.InOrder(
			mockRepo.EXPECT().SaveEmailLog(ctx, gomock.AssignableToTypeOf(model.EmailLog{})).Return(nil),
			mockClient.EXPECT().Send(ctx, "user@example.com", "Welcome", "Hello user!").Return(nil),
		)

		res, err := svc.SendEmail(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestTriggerEventEmail(t *testing.T) {
	t.Run("Success - Trigger Event Email", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.EventEmailRequest{
			EventType:     "payment.success",
			ReceiverEmail: "user@example.com",
			ReceiverID:    "user1",
			ReferenceID:   "pay-123",
		}

		gomock.InOrder(
			mockRepo.EXPECT().SaveEmailLog(ctx, gomock.AssignableToTypeOf(model.EmailLog{})).DoAndReturn(func(ctx context.Context, log model.EmailLog) error {
				assert.Equal(t, "Invoice", log.Subject) 
				assert.Equal(t, "user@example.com", log.ReceiverEmail)
				assert.Equal(t, "sent", log.Status)
				return nil
			}),
			mockClient.EXPECT().Send(ctx, "user@example.com", "Invoice", "Detail transaksi").Return(nil), 
		)

		res, err := svc.TriggerEventEmail(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "user@example.com", res.ReceiverEmail)
		assert.Equal(t, "Invoice", res.Subject)
		assert.Equal(t, "sent", res.Status)
		assert.NotEmpty(t, res.EmailID)
		assert.False(t, res.Timestamp.IsZero())
	})

	t.Run("Error - Invalid Event", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.EventEmailRequest{
			EventType:     "unknown.event",
			ReceiverEmail: "user@example.com",
		}

		mockRepo.EXPECT().SaveEmailLog(gomock.Any(), gomock.Any()).Times(0)
		mockClient.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.TriggerEventEmail(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "invalid event", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Repository gagal", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.EventEmailRequest{
			EventType:     "payment.success",
			ReceiverEmail: "user@example.com",
			ReceiverID:    "user1",
			ReferenceID:   "pay-123",
		}

		mockRepo.EXPECT().SaveEmailLog(ctx, gomock.Any()).Return(errors.New("db error"))
		mockClient.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.TriggerEventEmail(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		assert.Nil(t, res)
	})

	t.Run("Error - Sender gagal", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.EventEmailRequest{
			EventType:     "payment.success",
			ReceiverEmail: "user@example.com",
			ReceiverID:    "user1",
			ReferenceID:   "pay-123",
		}

		gomock.InOrder(
			mockRepo.EXPECT().SaveEmailLog(ctx, gomock.AssignableToTypeOf(model.EmailLog{})).DoAndReturn(func(ctx context.Context, log model.EmailLog) error {
				assert.Equal(t, "sent", log.Status)
				return nil
			}),
			mockClient.EXPECT().Send(ctx, "user@example.com", "Invoice", "Detail transaksi").Return(errors.New("smtp error")),
			mockRepo.EXPECT().SaveEmailLog(ctx, gomock.AssignableToTypeOf(model.EmailLog{})).DoAndReturn(func(ctx context.Context, log model.EmailLog) error {
				assert.Equal(t, "failed", log.Status)
				return nil
			}),
		)

		res, err := svc.TriggerEventEmail(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "smtp error", err.Error())
		assert.Nil(t, res)
	})
}

func TestSendEmailWithResult(t *testing.T) {
	t.Run("Success - With Result", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			Subject:       "Welcome",
			Body:          "Hello user!",
		}

		gomock.InOrder(
			mockRepo.EXPECT().SaveEmailLog(ctx, gomock.Any()).Return(nil),
			mockClient.EXPECT().Send(ctx, "user@example.com", "Welcome", "Hello user!").Return(nil),
		)

		res, err := svc.SendEmailWithResult(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "success", res.Status)
		assert.Equal(t, "email berhasil dikirim", res.Message)
		assert.NotNil(t, res.Data)
		assert.Equal(t, "user@example.com", res.Data.ReceiverEmail)
	})

	t.Run("Error - With Result", func(t *testing.T) {
		svc, mockRepo, mockClient, ctrl := newTestService(t)
		defer ctrl.Finish()

		ctx := context.Background()
		req := model.SendEmailRequest{
			ReceiverEmail: "user@example.com",
			Subject:       "Welcome",
			Body:          "Hello user!",
		}

		mockRepo.EXPECT().SaveEmailLog(ctx, gomock.Any()).Return(errors.New("db error"))
		mockClient.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		res, err := svc.SendEmailWithResult(ctx, req)

		assert.Error(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "failed", res.Status)
		assert.Equal(t, "db error", res.Message)
		assert.Nil(t, res.Data)
	})
}
