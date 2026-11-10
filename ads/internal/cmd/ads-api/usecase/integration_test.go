//go:build integration

package usecase_test

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/postgres"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

// These tests need the database of docker compose:
//
//	docker compose up -d ads-database
//	go test -tags integration ./...
//
// Every test uses a new user id, so the seed data and previous runs do not matter.

func newTestDB(t *testing.T) (*postgres.DAO, *sql.DB) {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://ads:ads@localhost:5432/ads?sslmode=disable"
	}

	dao, err := postgres.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = dao.Close() })

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return dao, db
}

func newUser(t *testing.T, db *sql.DB) domain.UserID {
	t.Helper()

	var id string
	require.NoError(t, db.QueryRow(`SELECT gen_random_uuid()::text`).Scan(&id))

	return domain.UserID(id)
}

func countAds(t *testing.T, dao *postgres.DAO, owner domain.UserID) int {
	t.Helper()

	ads, err := dao.SearchAds(context.Background(), adsdao.SearchAdsRequest{OwnerID: owner})
	require.NoError(t, err)

	return len(ads)
}

func TestIntegration_Quota(t *testing.T) {
	ctx := context.Background()
	dao, db := newTestDB(t)
	createAd := usecase.NewCreateAdFunc(dao)

	t.Run("10 concurrent requests publish 5 ads", func(t *testing.T) {
		owner := newUser(t, db)
		errs := make(chan error, 10)

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				_, err := createAd(ctx, usecase.CreateAdRequest{Title: "bike", PriceCents: 5000, OwnerID: owner})
				errs <- err
			})
		}
		wg.Wait()
		close(errs)

		rejected := 0
		for err := range errs {
			if err != nil {
				assert.ErrorIs(t, err, usecase.ErrQuotaExceeded)
				rejected++
			}
		}

		assert.Equal(t, 5, rejected)
		assert.Equal(t, 5, countAds(t, dao, owner))
	})

	t.Run("a publication older than 24 hours no longer counts", func(t *testing.T) {
		owner := newUser(t, db)
		for range 5 {
			_, err := createAd(ctx, usecase.CreateAdRequest{Title: "bike", PriceCents: 5000, OwnerID: owner})
			require.NoError(t, err)
		}

		_, err := createAd(ctx, usecase.CreateAdRequest{Title: "bike", PriceCents: 5000, OwnerID: owner})
		require.ErrorIs(t, err, usecase.ErrQuotaExceeded)

		_, err = db.Exec(`
			UPDATE ad_publications SET published_at = now() - interval '25 hours'
			WHERE ad_id = (SELECT min(ad_id) FROM ad_publications WHERE user_id = $1::uuid)`, string(owner))
		require.NoError(t, err)

		_, err = createAd(ctx, usecase.CreateAdRequest{Title: "bike", PriceCents: 5000, OwnerID: owner})
		assert.NoError(t, err)
	})

	t.Run("imported ads do not count", func(t *testing.T) {
		owner := newUser(t, db)
		ads := make([]usecase.CreateAdRequest, 10)
		for i := range ads {
			ads[i] = usecase.CreateAdRequest{Title: "bike", PriceCents: 5000, OwnerID: owner}
		}

		_, err := usecase.NewImportAdsFunc(dao)(ctx, ads)
		require.NoError(t, err)

		_, err = createAd(ctx, usecase.CreateAdRequest{Title: "bike", PriceCents: 5000, OwnerID: owner})
		assert.NoError(t, err)
	})
}

func TestIntegration_ImportIsAllOrNothing(t *testing.T) {
	dao, db := newTestDB(t)
	owner := newUser(t, db)

	// PostgreSQL refuses a NUL byte in a text column, so the third insert fails after two that worked.
	_, err := usecase.NewImportAdsFunc(dao)(context.Background(), []usecase.CreateAdRequest{
		{Title: "bike", PriceCents: 5000, OwnerID: owner},
		{Title: "scooter", PriceCents: 8000, OwnerID: owner},
		{Title: "car\x00", PriceCents: 200000, OwnerID: owner},
	})

	require.Error(t, err)
	assert.Zero(t, countAds(t, dao, owner))
}
