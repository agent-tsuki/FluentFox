package humautil

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/fluentfox/api/pkg/exceptions"
	"go.uber.org/zap"
)

// MapErr converts a domain error into a huma HTTP error.
//
// AppErrors carry their own HTTP status code and message and are surfaced
// directly to the caller. Anything else is logged as an internal error and
// returns a generic 500 so implementation details are never leaked.
func MapErr(err error, log *zap.Logger) error {
	if appErr, ok := exceptions.As(err); ok {
		return huma.NewError(appErr.Status, appErr.Message)
	}
	log.Error("unhandled internal error", zap.Error(err))
	return huma.NewError(http.StatusInternalServerError, "an unexpected error occurred")
}
