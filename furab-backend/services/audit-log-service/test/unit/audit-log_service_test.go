package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/audit-log-service/internal/model"
	"furab-backend/services/audit-log-service/internal/service"
	"furab-backend/services/audit-log-service/test/unit/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuditLogService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockAuditLogRepository(ctrl)
	svc := service.NewAuditLogService(mockRepo)
	ctx := context.Background()

	t.Run("CreateAuditLog_Success", func(t *testing.T) {
		m := &model.AuditLog{UserID: "u-123", Action: "CREATE", Entity: "USER", EntityID: "u-123"}
		mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(nil)

		err := svc.CreateAuditLog(ctx, m)
		assert.NoError(t, err)
		assert.NotEmpty(t, m.ID)
		assert.NotZero(t, m.CreatedAt)
	})

	t.Run("CreateAuditLog_Error_MissingUserID", func(t *testing.T) {
		m := &model.AuditLog{Action: "CREATE", Entity: "USER"}
		err := svc.CreateAuditLog(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "user id is required", err.Error())
	})

	t.Run("CreateAuditLog_Error_MissingAction", func(t *testing.T) {
		m := &model.AuditLog{UserID: "u-123", Entity: "USER"}
		err := svc.CreateAuditLog(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "action is required", err.Error())
	})

	t.Run("CreateAuditLog_Error_MissingEntity", func(t *testing.T) {
		m := &model.AuditLog{UserID: "u-123", Action: "CREATE"}
		err := svc.CreateAuditLog(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "entity is required", err.Error())
	})

	t.Run("CreateAuditLog_Error_RepoSave", func(t *testing.T) {
		m := &model.AuditLog{UserID: "u-123", Action: "CREATE", Entity: "USER"}
		mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(errors.New("db error"))

		err := svc.CreateAuditLog(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})

	t.Run("GetAuditLog_Success", func(t *testing.T) {
		id := "log-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(&model.AuditLog{ID: id, Action: "CREATE"}, nil)

		m, err := svc.GetAuditLog(ctx, id)
		assert.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, "CREATE", m.Action)
	})

	t.Run("GetAuditLog_Error_EmptyID", func(t *testing.T) {
		m, err := svc.GetAuditLog(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, m)
		assert.Equal(t, "audit log id is required", err.Error())
	})

	t.Run("UpdateAuditLog_Success", func(t *testing.T) {
		id := "log-123"
		existing := &model.AuditLog{ID: id, Action: "CREATE"}
		updateReq := &model.AuditLog{ID: id, Action: "UPDATE"}

		mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		err := svc.UpdateAuditLog(ctx, updateReq)
		assert.NoError(t, err)
	})

	t.Run("UpdateAuditLog_Error_NotFound", func(t *testing.T) {
		id := "log-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(nil, errors.New("not found"))

		err := svc.UpdateAuditLog(ctx, &model.AuditLog{ID: id})
		assert.Error(t, err)
	})

	t.Run("DeleteAuditLog_Success", func(t *testing.T) {
		id := "log-123"
		mockRepo.EXPECT().Delete(ctx, id).Return(nil)

		err := svc.DeleteAuditLog(ctx, id)
		assert.NoError(t, err)
	})

	t.Run("SearchAuditLogs_Success", func(t *testing.T) {
		req := model.SearchAuditLogRequest{UserID: "u-123", Limit: 10, Offset: 0}
		mockRepo.EXPECT().Search(ctx, req).Return([]model.AuditLog{{Action: "CREATE"}}, 1, nil)

		res, total, err := svc.SearchAuditLogs(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, 1, total)
	})
}
