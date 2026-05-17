package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/merchant-service/internal/model"
	"furab-backend/services/merchant-service/internal/service"
	"furab-backend/services/merchant-service/test/unit/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMerchantService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockMerchantRepository(ctrl)
	svc := service.NewMerchantService(mockRepo)
	ctx := context.Background()

	t.Run("CreateMerchant_Success", func(t *testing.T) {
		m := &model.Merchant{Name: "Bakso Beranak", Address: "Jl. Mawar No. 1"}
		mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(nil)

		err := svc.CreateMerchant(ctx, m)
		assert.NoError(t, err)
		assert.NotEmpty(t, m.ID)
		assert.True(t, m.IsOpen)
	})

	t.Run("CreateMerchant_Error_MissingName", func(t *testing.T) {
		m := &model.Merchant{Address: "Jl. Mawar No. 1"}
		err := svc.CreateMerchant(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "merchant name is required", err.Error())
	})

	t.Run("CreateMerchant_Error_MissingAddress", func(t *testing.T) {
		m := &model.Merchant{Name: "Bakso Beranak"}
		err := svc.CreateMerchant(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "merchant address is required", err.Error())
	})

	t.Run("CreateMerchant_Error_RepoSave", func(t *testing.T) {
		m := &model.Merchant{Name: "Bakso Beranak", Address: "Jl. Mawar No. 1"}
		mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(errors.New("db error"))

		err := svc.CreateMerchant(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})

	t.Run("GetMerchant_Success", func(t *testing.T) {
		id := "uuid-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(&model.Merchant{ID: id, Name: "Test"}, nil)

		m, err := svc.GetMerchant(ctx, id)
		assert.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, "Test", m.Name)
	})

	t.Run("GetMerchant_Error_EmptyID", func(t *testing.T) {
		m, err := svc.GetMerchant(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, m)
		assert.Equal(t, "merchant id is required", err.Error())
	})

	t.Run("GetMerchant_Error_NotFound", func(t *testing.T) {
		id := "uuid-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(nil, errors.New("not found"))

		m, err := svc.GetMerchant(ctx, id)
		assert.Error(t, err)
		assert.Nil(t, m)
		assert.Equal(t, "not found", err.Error())
	})

	t.Run("UpdateMerchant_Success", func(t *testing.T) {
		id := "uuid-123"
		existing := &model.Merchant{ID: id, Name: "Old Name"}
		updateReq := &model.Merchant{ID: id, Name: "New Name"}

		mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		err := svc.UpdateMerchant(ctx, updateReq)
		assert.NoError(t, err)
	})

	t.Run("UpdateMerchant_Error_GetByID", func(t *testing.T) {
		id := "uuid-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(nil, errors.New("db error"))

		err := svc.UpdateMerchant(ctx, &model.Merchant{ID: id})
		assert.Error(t, err)
	})

	t.Run("DeleteMerchant_Success", func(t *testing.T) {
		id := "uuid-123"
		mockRepo.EXPECT().Delete(ctx, id).Return(nil)

		err := svc.DeleteMerchant(ctx, id)
		assert.NoError(t, err)
	})

	t.Run("SearchMerchants_Success", func(t *testing.T) {
		req := model.SearchMerchantRequest{Query: "Bakso", Limit: 10, Offset: 0}
		mockRepo.EXPECT().Search(ctx, req).Return([]model.Merchant{{Name: "Bakso Beranak"}}, 1, nil)

		res, total, err := svc.SearchMerchants(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, 1, total)
	})
}
