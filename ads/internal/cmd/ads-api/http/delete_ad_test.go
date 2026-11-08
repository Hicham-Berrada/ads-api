package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
)

func newDeleteAdServer(fn usecase.DeleteAdFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("DELETE /ads/{adID}", middleware.RequireAuth(handlerDeleteAd(fn)))

	return httptest.NewServer(mux)
}

func deleteAd(t *testing.T, url, authHeader string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

func TestHandlerDeleteAd(t *testing.T) {
	t.Run("returns 204 and passes the ad id and the authenticated owner", func(t *testing.T) {
		var received usecase.DeleteAdRequest
		srv := newDeleteAdServer(func(_ context.Context, req usecase.DeleteAdRequest) error {
			received = req
			return nil
		})
		defer srv.Close()

		resp := deleteAd(t, srv.URL+"/ads/42", "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Equal(t, usecase.DeleteAdRequest{ID: 42, OwnerID: "user-1"}, received)
	})

	t.Run("returns 401 when Authorization header is missing", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ usecase.DeleteAdRequest) error {
			t.Fatal("usecase should not be called")
			return nil
		})
		defer srv.Close()

		resp := deleteAd(t, srv.URL+"/ads/42", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("returns 400 when adID is not a number", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ usecase.DeleteAdRequest) error {
			t.Fatal("usecase should not be called")
			return nil
		})
		defer srv.Close()

		resp := deleteAd(t, srv.URL+"/ads/not-a-number", "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 404 when the ad does not exist or belongs to someone else", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ usecase.DeleteAdRequest) error {
			return adsdao.ErrNotFound
		})
		defer srv.Close()

		resp := deleteAd(t, srv.URL+"/ads/42", "user-2")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 500 on unexpected usecase error", func(t *testing.T) {
		srv := newDeleteAdServer(func(_ context.Context, _ usecase.DeleteAdRequest) error {
			return errors.New("db down")
		})
		defer srv.Close()

		resp := deleteAd(t, srv.URL+"/ads/42", "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}
