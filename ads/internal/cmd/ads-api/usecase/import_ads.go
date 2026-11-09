package usecase

import (
	"context"
	"fmt"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

type (
	// ImportAdsFunc creates several ads at once. Either all of them are created, or none.
	ImportAdsFunc func(ctx context.Context, ads []CreateAdRequest) ([]domain.AdID, error)

	importAds struct {
		adDAO adsdao.DAO
	}
)

// NewImportAdsFunc returns an ImportAdsFunc backed by the given DAO.
func NewImportAdsFunc(adDAO adsdao.DAO) ImportAdsFunc {
	return importAds{adDAO: adDAO}.importAds
}

func (uc importAds) importAds(ctx context.Context, ads []CreateAdRequest) ([]domain.AdID, error) {
	// One transaction for the whole import: if an ad fails, nothing is kept and the import can be sent again as is.
	tx, err := uc.adDAO.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to import ads: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	adIDs := make([]domain.AdID, 0, len(ads))

	for _, ad := range ads {
		adID, err := tx.CreateAd(ctx, adsdao.CreateAdRequest{
			Title:      ad.Title,
			PriceCents: ad.PriceCents,
			PhotoURL:   ad.PhotoURL,
			OwnerID:    ad.OwnerID,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to import ads: %w", err)
		}

		adIDs = append(adIDs, adID)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("unable to import ads: %w", err)
	}

	return adIDs, nil
}
