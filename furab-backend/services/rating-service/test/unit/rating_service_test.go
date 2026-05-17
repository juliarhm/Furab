package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/rating-service/internal/model"
	"furab-backend/services/rating-service/internal/service"
	"furab-backend/services/rating-service/test/unit/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRatingService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockRatingRepository(ctrl)
	svc := service.NewRatingService(mockRepo)
	ctx := context.Background()

	t.Run("CreateRating_Success", func(t *testing.T) {
		m := &model.Rating{UserID: "u-123", TargetID: "t-123", TargetType: "merchant", Score: 5, Comment: "Great!"}
		mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(nil)

		err := svc.CreateRating(ctx, m)
		assert.NoError(t, err)
		assert.NotEmpty(t, m.ID)
	})

	t.Run("CreateRating_Error_MissingUserID", func(t *testing.T) {
		m := &model.Rating{TargetID: "t-123", TargetType: "merchant", Score: 5}
		err := svc.CreateRating(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "user id is required", err.Error())
	})

	t.Run("CreateRating_Error_MissingTargetID", func(t *testing.T) {
		m := &model.Rating{UserID: "u-123", TargetType: "merchant", Score: 5}
		err := svc.CreateRating(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "target id is required", err.Error())
	})

	t.Run("CreateRating_Error_MissingTargetType", func(t *testing.T) {
		m := &model.Rating{UserID: "u-123", TargetID: "t-123", Score: 5}
		err := svc.CreateRating(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "target type is required", err.Error())
	})

	t.Run("CreateRating_Error_InvalidScoreLow", func(t *testing.T) {
		m := &model.Rating{UserID: "u-123", TargetID: "t-123", TargetType: "merchant", Score: 0}
		err := svc.CreateRating(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "score must be between 1 and 5", err.Error())
	})

	t.Run("CreateRating_Error_InvalidScoreHigh", func(t *testing.T) {
		m := &model.Rating{UserID: "u-123", TargetID: "t-123", TargetType: "merchant", Score: 6}
		err := svc.CreateRating(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "score must be between 1 and 5", err.Error())
	})

	t.Run("GetRating_Success", func(t *testing.T) {
		id := "rating-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(&model.Rating{ID: id, Score: 5}, nil)

		m, err := svc.GetRating(ctx, id)
		assert.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, 5, m.Score)
	})

	t.Run("GetRating_Error_EmptyID", func(t *testing.T) {
		m, err := svc.GetRating(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, m)
		assert.Equal(t, "rating id is required", err.Error())
	})

	t.Run("UpdateRating_Success", func(t *testing.T) {
		id := "rating-123"
		existing := &model.Rating{ID: id, Score: 4}
		updateReq := &model.Rating{ID: id, Score: 5, Comment: "Updated"}

		mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		err := svc.UpdateRating(ctx, updateReq)
		assert.NoError(t, err)
	})

	t.Run("UpdateRating_Error_NotFound", func(t *testing.T) {
		id := "rating-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(nil, errors.New("not found"))

		err := svc.UpdateRating(ctx, &model.Rating{ID: id, Score: 5})
		assert.Error(t, err)
	})

	t.Run("DeleteRating_Success", func(t *testing.T) {
		id := "rating-123"
		mockRepo.EXPECT().Delete(ctx, id).Return(nil)

		err := svc.DeleteRating(ctx, id)
		assert.NoError(t, err)
	})

	t.Run("SearchRatings_Success", func(t *testing.T) {
		req := model.SearchRatingRequest{TargetID: "t-123", TargetType: "merchant", Limit: 10, Offset: 0}
		mockRepo.EXPECT().Search(ctx, req).Return([]model.Rating{{Score: 5}}, 1, nil)

		res, total, err := svc.SearchRatings(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, 1, total)
	})
}
