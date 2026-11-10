package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

const (
	// MaxPublishedAds is how many ads a user can publish within QuotaPeriod.
	MaxPublishedAds = 5
	// QuotaPeriod is the rolling period of the publication quota.
	QuotaPeriod = 24 * time.Hour
)

// ErrQuotaExceeded is returned when the user already published MaxPublishedAds ads within QuotaPeriod.
var ErrQuotaExceeded = errors.New("publication quota exceeded")

type (
	// CreateAdFunc creates an ad owned by the calling user. It fails with ErrQuotaExceeded when the user is over the publication quota.
	CreateAdFunc func(ctx context.Context, req CreateAdRequest) (domain.AdID, error)

	// CreateAdRequest describes the ad to create. OwnerID is the authenticated user: a caller can never create an ad on behalf of someone else.
	CreateAdRequest struct {
		Title      string
		PriceCents int64
		PhotoURL   string
		OwnerID    domain.UserID
	}

	createAd struct {
		adDAO adsdao.DAO
	}
)

// NewCreateAdFunc returns a CreateAdFunc backed by the given DAO.
func NewCreateAdFunc(adDAO adsdao.DAO) CreateAdFunc {
	return createAd{adDAO: adDAO}.createAd
}

func (uc createAd) createAd(ctx context.Context, req CreateAdRequest) (domain.AdID, error) {
	tx, err := uc.adDAO.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Without the lock, two requests of the same user could both count 4 publications and both publish a 5th one.
	if err := tx.LockOwner(ctx, req.OwnerID); err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	published, err := tx.CountPublications(ctx, req.OwnerID, QuotaPeriod)
	if err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	if published >= MaxPublishedAds {
		return 0, ErrQuotaExceeded
	}

	adID, err := tx.CreateAd(ctx, adsdao.CreateAdRequest{
		Title:      req.Title,
		PriceCents: req.PriceCents,
		PhotoURL:   req.PhotoURL,
		OwnerID:    req.OwnerID,
	})
	if err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	if err := tx.CreatePublication(ctx, adID, req.OwnerID); err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	return adID, nil
}
