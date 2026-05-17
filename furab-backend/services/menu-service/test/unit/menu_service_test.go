package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/menu-service/internal/model"
	"furab-backend/services/menu-service/internal/service"
	"furab-backend/services/menu-service/test/unit/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMenuService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockMenuRepository(ctrl)
	svc := service.NewMenuService(mockRepo)
	ctx := context.Background()

	t.Run("CreateMenu_Success", func(t *testing.T) {
		m := &model.Menu{MerchantID: "m-123", Name: "Nasi Goreng", Price: 20000}
		mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(nil)

		err := svc.CreateMenu(ctx, m)
		assert.NoError(t, err)
		assert.NotEmpty(t, m.ID)
		assert.True(t, m.IsAvailable)
	})

	t.Run("CreateMenu_Error_MissingMerchantID", func(t *testing.T) {
		m := &model.Menu{Name: "Nasi Goreng", Price: 20000}
		err := svc.CreateMenu(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "merchant id is required", err.Error())
	})

	t.Run("CreateMenu_Error_MissingName", func(t *testing.T) {
		m := &model.Menu{MerchantID: "m-123", Price: 20000}
		err := svc.CreateMenu(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "menu name is required", err.Error())
	})

	t.Run("CreateMenu_Error_InvalidPrice", func(t *testing.T) {
		m := &model.Menu{MerchantID: "m-123", Name: "Nasi Goreng", Price: 0}
		err := svc.CreateMenu(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "price must be greater than zero", err.Error())
	})

	t.Run("CreateMenu_Error_RepoSave", func(t *testing.T) {
		m := &model.Menu{MerchantID: "m-123", Name: "Nasi Goreng", Price: 20000}
		mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(errors.New("db error"))

		err := svc.CreateMenu(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})

	t.Run("GetMenu_Success", func(t *testing.T) {
		id := "menu-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(&model.Menu{ID: id, Name: "Nasi Goreng"}, nil)

		m, err := svc.GetMenu(ctx, id)
		assert.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, "Nasi Goreng", m.Name)
	})

	t.Run("GetMenu_Error_EmptyID", func(t *testing.T) {
		m, err := svc.GetMenu(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, m)
		assert.Equal(t, "menu id is required", err.Error())
	})

	t.Run("GetMenu_Error_NotFound", func(t *testing.T) {
		id := "menu-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(nil, errors.New("not found"))

		m, err := svc.GetMenu(ctx, id)
		assert.Error(t, err)
		assert.Nil(t, m)
		assert.Equal(t, "not found", err.Error())
	})

	t.Run("UpdateMenu_Success", func(t *testing.T) {
		id := "menu-123"
		existing := &model.Menu{ID: id, Name: "Old Name"}
		updateReq := &model.Menu{ID: id, Name: "New Name"}

		mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		err := svc.UpdateMenu(ctx, updateReq)
		assert.NoError(t, err)
	})

	t.Run("UpdateMenu_Error_GetByID", func(t *testing.T) {
		id := "menu-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(nil, errors.New("db error"))

		err := svc.UpdateMenu(ctx, &model.Menu{ID: id})
		assert.Error(t, err)
	})

	t.Run("DeleteMenu_Success", func(t *testing.T) {
		id := "menu-123"
		mockRepo.EXPECT().Delete(ctx, id).Return(nil)

		err := svc.DeleteMenu(ctx, id)
		assert.NoError(t, err)
	})

	t.Run("SearchMenus_Success", func(t *testing.T) {
		req := model.SearchMenuRequest{MerchantID: "m-123", Query: "Nasi", Limit: 10, Offset: 0}
		mockRepo.EXPECT().Search(ctx, req).Return([]model.Menu{{Name: "Nasi Goreng"}}, 1, nil)

		res, total, err := svc.SearchMenus(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, 1, total)
	})
}
