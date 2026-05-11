//go:build wireinject

//go:generate go run -mod=mod github.com/google/wire/cmd/wire

package di

import (
	"net/http"

	"github.com/google/wire"

	"ads-service/internal/application/usecases"
	"ads-service/internal/domain/ports"
	"ads-service/internal/infrastructure/adapters/postgres"
	httpapi "ads-service/internal/presentation/http"
)

func InitializeHTTPHandler() (http.Handler, error) {
	wire.Build(
		postgres.OpenDB,
		postgres.NewAdsRepository,
		wire.Bind(new(ports.AdsRepository), new(*postgres.AdsRepository)),
		usecases.NewCreateAdUseCase,
		usecases.NewPublishAdUseCase,
		usecases.NewGetAdUseCase,
		httpapi.NewRouter,
	)
	return nil, nil
}
