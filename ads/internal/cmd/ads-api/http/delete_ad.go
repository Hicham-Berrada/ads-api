package http

import (
	"errors"
	"net/http"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
)

// handlerDeleteAd serves DELETE /ads/{adID}. The ad of another user answers 404, like a missing one.
func handlerDeleteAd(deleteAd usecase.DeleteAdFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		adID, err := adIDFromURL(r)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)

			return
		}

		err = deleteAd(r.Context(), usecase.DeleteAdRequest{
			ID:      adID,
			OwnerID: middleware.UserIDFromRequest(r),
		})
		if err != nil {
			if errors.Is(err, adsdao.ErrNotFound) {
				rw.WriteHeader(http.StatusNotFound)
			} else {
				rw.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		rw.WriteHeader(http.StatusNoContent)
	}
}
