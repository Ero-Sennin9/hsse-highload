//go:build wireinject

//go:generate go run -mod=mod github.com/google/wire/cmd/wire

package di

import (
	"net/http"

	"github.com/google/wire"

	"billing-service/internal/application/usecases"
	"billing-service/internal/domain/ports"
	httpadapter "billing-service/internal/infrastructure/adapters/http"
	"billing-service/internal/infrastructure/adapters/postgres"
	httpapi "billing-service/internal/presentation/http"
)

func InitializeHTTPHandler() (http.Handler, error) {
	wire.Build(
		provideHTTPClient,
		wire.Bind(new(ports.HTTPClient), new(*http.Client)),
		httpadapter.NewAdsCatalogAdapter,
		wire.Bind(new(ports.AdsCatalog), new(*httpadapter.AdsCatalogAdapter)),
		postgres.OpenDB,
		postgres.NewPromotionsRepository,
		wire.Bind(new(ports.PromotionsRepository), new(*postgres.PromotionsRepository)),
		usecases.NewPurchasePromotionUseCase,
		httpapi.NewRouter,
	)
	return nil, nil
}
