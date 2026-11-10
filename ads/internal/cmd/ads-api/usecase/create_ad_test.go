package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

func TestCreateAd(t *testing.T) {
	ctx := context.Background()
	owner := domain.UserID("user-1")

	ucReq := usecase.CreateAdRequest{
		Title:      "mountain bike",
		PriceCents: 15000,
		PhotoURL:   "https://example.com/bike.jpg",
		OwnerID:    owner,
	}
	daoReq := adsdao.CreateAdRequest{
		Title:      "mountain bike",
		PriceCents: 15000,
		PhotoURL:   "https://example.com/bike.jpg",
		OwnerID:    owner,
	}

	t.Run("nominal", func(t *testing.T) {
		m := mock.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwner", ctx, owner).Return(nil)
		m.On("CountPublications", ctx, owner, 24*time.Hour).Return(4, nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(7), nil)
		m.On("CreatePublication", ctx, domain.AdID(7), owner).Return(nil)
		m.On("Commit").Return(nil)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		require.NoError(t, err)
		assert.Equal(t, domain.AdID(7), got)
	})

	t.Run("error quota exceeded", func(t *testing.T) {
		m := mock.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwner", ctx, owner).Return(nil)
		m.On("CountPublications", ctx, owner, 24*time.Hour).Return(5, nil)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		assert.ErrorIs(t, err, usecase.ErrQuotaExceeded)
		m.AssertNotCalled(t, "CreateAd")
	})

	t.Run("error from dao", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwner", ctx, owner).Return(nil)
		m.On("CountPublications", ctx, owner, 24*time.Hour).Return(0, nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(0), dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		require.Error(t, err)
		assert.ErrorIs(t, err, dbErr)
		m.AssertNotCalled(t, "Commit")
	})
}
