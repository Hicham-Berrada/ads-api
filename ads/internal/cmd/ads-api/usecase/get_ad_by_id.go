package usecase

import (
	"context"
	"fmt"
	"log/slog"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
)

type (
	// GetAdByIDFunc returns a single ad and the nickname of its owner. It fails with dao.ErrNotFound when no ad matches the identifier.
	GetAdByIDFunc func(ctx context.Context, id domain.AdID) (GetAdByIDResponse, error)

	// GetAdByIDResponse is an ad and the nickname of its owner. OwnerNickname is empty when the user service could not provide it.
	GetAdByIDResponse struct {
		Ad            domain.Ad
		OwnerNickname string
	}

	getAdByID struct {
		adDAO      adsdao.DAO
		userClient usersclient.UserClient
	}
)

// NewGetAdByIDFunc returns a GetAdByIDFunc backed by the given DAO and user service client.
func NewGetAdByIDFunc(adDAO adsdao.DAO, userClient usersclient.UserClient) GetAdByIDFunc {
	return getAdByID{adDAO: adDAO, userClient: userClient}.getAdByID
}

func (uc getAdByID) getAdByID(ctx context.Context, id domain.AdID) (GetAdByIDResponse, error) {
	found, err := uc.adDAO.GetAdByID(ctx, id)
	if err != nil {
		return GetAdByIDResponse{}, fmt.Errorf("unable to get ad %s: %w", id, err)
	}

	owner, err := uc.userClient.GetUserByID(ctx, found.UserID)
	if err != nil {
		// Reads must not depend on the user service: the ad is returned without the nickname.
		slog.WarnContext(ctx, "unable to get the owner of an ad", "ad_id", id, "user_id", found.UserID, "error", err)

		return GetAdByIDResponse{Ad: found}, nil
	}

	return GetAdByIDResponse{Ad: found, OwnerNickname: owner.Nickname}, nil
}
