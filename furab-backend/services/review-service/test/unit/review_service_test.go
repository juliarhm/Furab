package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/review-service/internal/model"
	"furab-backend/services/review-service/internal/service"
	"furab-backend/services/review-service/test/unit/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestReviewService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockReviewRepository(ctrl)
	svc := service.NewReviewService(mockRepo)
	ctx := context.Background()

	t.Run("CreateReview_Success", func(t *testing.T) {
		m := &model.Review{UserID: "u-123", MerchantID: "m-123", OrderID: "o-123", Rating: 5, Comment: "Excellent!"}
		mockRepo.EXPECT().Save(ctx, gomock.Any()).Return(nil)

		err := svc.CreateReview(ctx, m)
		assert.NoError(t, err)
		assert.NotEmpty(t, m.ID)
	})

	t.Run("CreateReview_Error_MissingUserID", func(t *testing.T) {
		m := &model.Review{MerchantID: "m-123", OrderID: "o-123", Rating: 5}
		err := svc.CreateReview(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "user id is required", err.Error())
	})

	t.Run("CreateReview_Error_MissingMerchantID", func(t *testing.T) {
		m := &model.Review{UserID: "u-123", OrderID: "o-123", Rating: 5}
		err := svc.CreateReview(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "merchant id is required", err.Error())
	})

	t.Run("CreateReview_Error_MissingOrderID", func(t *testing.T) {
		m := &model.Review{UserID: "u-123", MerchantID: "m-123", Rating: 5}
		err := svc.CreateReview(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "order id is required", err.Error())
	})

	t.Run("CreateReview_Error_InvalidRatingLow", func(t *testing.T) {
		m := &model.Review{UserID: "u-123", MerchantID: "m-123", OrderID: "o-123", Rating: 0}
		err := svc.CreateReview(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "rating must be between 1 and 5", err.Error())
	})

	t.Run("CreateReview_Error_InvalidRatingHigh", func(t *testing.T) {
		m := &model.Review{UserID: "u-123", MerchantID: "m-123", OrderID: "o-123", Rating: 6}
		err := svc.CreateReview(ctx, m)
		assert.Error(t, err)
		assert.Equal(t, "rating must be between 1 and 5", err.Error())
	})

	t.Run("GetReview_Success", func(t *testing.T) {
		id := "review-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(&model.Review{ID: id, Rating: 5}, nil)

		m, err := svc.GetReview(ctx, id)
		assert.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, 5, m.Rating)
	})

	t.Run("GetReview_Error_EmptyID", func(t *testing.T) {
		m, err := svc.GetReview(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, m)
		assert.Equal(t, "review id is required", err.Error())
	})

	t.Run("UpdateReview_Success", func(t *testing.T) {
		id := "review-123"
		existing := &model.Review{ID: id, Rating: 4}
		updateReq := &model.Review{ID: id, Rating: 5, Comment: "Updated"}

		mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
		mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		err := svc.UpdateReview(ctx, updateReq)
		assert.NoError(t, err)
	})

	t.Run("UpdateReview_Error_NotFound", func(t *testing.T) {
		id := "review-123"
		mockRepo.EXPECT().GetByID(ctx, id).Return(nil, errors.New("not found"))

		err := svc.UpdateReview(ctx, &model.Review{ID: id, Rating: 5})
		assert.Error(t, err)
	})

	t.Run("DeleteReview_Success", func(t *testing.T) {
		id := "review-123"
		mockRepo.EXPECT().Delete(ctx, id).Return(nil)

		err := svc.DeleteReview(ctx, id)
		assert.NoError(t, err)
	})

	t.Run("SearchReviews_Success", func(t *testing.T) {
		req := model.SearchReviewRequest{MerchantID: "m-123", Limit: 10, Offset: 0}
		mockRepo.EXPECT().Search(ctx, req).Return([]model.Review{{Rating: 5}}, 1, nil)

		res, total, err := svc.SearchReviews(ctx, req)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, 1, total)
	})
}
