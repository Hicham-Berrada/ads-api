package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	mockdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

func TestImportAds(t *testing.T) {
	ctx := context.Background()

	ads := []usecase.CreateAdRequest{
		{Title: "bike", PriceCents: 5000, PhotoURL: "https://example.com/bike.jpg", OwnerID: "user-1"},
		{Title: "car", PriceCents: 200000, PhotoURL: "https://example.com/car.jpg", OwnerID: "user-1"},
	}
	createBike := adsdao.CreateAdRequest{Title: "bike", PriceCents: 5000, PhotoURL: "https://example.com/bike.jpg", OwnerID: "user-1"}
	createCar := adsdao.CreateAdRequest{Title: "car", PriceCents: 200000, PhotoURL: "https://example.com/car.jpg", OwnerID: "user-1"}

	t.Run("nominal", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("CreateAd", ctx, createBike).Return(domain.AdID(10), nil)
		m.On("CreateAd", ctx, createCar).Return(domain.AdID(11), nil)
		m.On("Commit").Return(nil)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewImportAdsFunc(m)(ctx, ads)

		require.NoError(t, err)
		assert.Equal(t, []domain.AdID{10, 11}, got)
	})

	t.Run("rolls back when an ad fails", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("CreateAd", ctx, createBike).Return(domain.AdID(10), nil)
		m.On("CreateAd", ctx, createCar).Return(domain.AdID(0), dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewImportAdsFunc(m)(ctx, ads)

		assert.ErrorIs(t, err, dbErr)
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("error when the transaction cannot be opened", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("too many connections")
		m.On("Begin", ctx).Return(dbErr)

		_, err := usecase.NewImportAdsFunc(m)(ctx, ads)

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("error on commit", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("CreateAd", ctx, createBike).Return(domain.AdID(10), nil)
		m.On("CreateAd", ctx, createCar).Return(domain.AdID(11), nil)
		m.On("Commit").Return(dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewImportAdsFunc(m)(ctx, ads)

		assert.ErrorIs(t, err, dbErr)
	})
}
