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
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	mockusers "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/mock"
)

func TestGetAdByID(t *testing.T) {
	ctx := context.Background()
	ad := domain.Ad{ID: 42, Title: "bike", Price: 5000, UserID: "user-1"}

	t.Run("nominal", func(t *testing.T) {
		m := mockdao.New(t)
		users := mockusers.New(t)
		m.On("GetAdByID", ctx, domain.AdID(42)).Return(ad, nil)
		users.On("GetUserByID", ctx, domain.UserID("user-1")).Return(usersclient.User{UserID: "user-1", Nickname: "alice"}, nil)

		got, err := usecase.NewGetAdByIDFunc(m, users)(ctx, 42)

		require.NoError(t, err)
		assert.Equal(t, usecase.GetAdByIDResponse{Ad: ad, OwnerNickname: "alice"}, got)
	})

	t.Run("returns the ad without nickname when the user service fails", func(t *testing.T) {
		m := mockdao.New(t)
		users := mockusers.New(t)
		m.On("GetAdByID", ctx, domain.AdID(42)).Return(ad, nil)
		users.On("GetUserByID", ctx, domain.UserID("user-1")).Return(usersclient.User{}, errors.New("identity service answered 503"))

		got, err := usecase.NewGetAdByIDFunc(m, users)(ctx, 42)

		require.NoError(t, err)
		assert.Equal(t, usecase.GetAdByIDResponse{Ad: ad}, got)
	})

	t.Run("error no ad found", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("GetAdByID", ctx, domain.AdID(99)).Return(domain.Ad{}, adsdao.ErrNotFound)

		_, err := usecase.NewGetAdByIDFunc(m, mockusers.New(t))(ctx, 99)

		require.Error(t, err)
		assert.ErrorIs(t, err, adsdao.ErrNotFound)
	})

	t.Run("error from dao", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("connection reset")
		m.On("GetAdByID", ctx, domain.AdID(1)).Return(domain.Ad{}, dbErr)

		_, err := usecase.NewGetAdByIDFunc(m, mockusers.New(t))(ctx, 1)

		assert.ErrorIs(t, err, dbErr)
	})
}
