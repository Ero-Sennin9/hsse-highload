//go:build wireinject

//go:generate go run -mod=mod github.com/google/wire/cmd/wire

package di

import (
	"net/http"

	"github.com/google/wire"

	"ads-service/internal/application/usecases"
	"ads-service/internal/domain/ports"
	"ads-service/internal/infrastructure/adapters/postgres"
	"ads-service/internal/infrastructure/adapters/s3"
	httpapi "ads-service/internal/presentation/http"
)

func InitializeHTTPHandler() (http.Handler, error) {
	wire.Build(
		postgres.OpenDB,
		postgres.NewAdsRepository,
		wire.Bind(new(ports.AdsRepository), new(*postgres.AdsRepository)),
		postgres.NewMediaRepository,
		wire.Bind(new(ports.MediaRepository), new(*postgres.MediaRepository)),
		s3.NewObjectStorage,
		wire.Bind(new(ports.ObjectStorage), new(*s3.ObjectStorage)),
		usecases.NewCreateAdUseCase,
		usecases.NewPublishAdUseCase,
		usecases.NewGetAdUseCase,
		usecases.NewUploadAdPhotoUseCase,
		httpapi.NewRouter,
	)
	return nil, nil
}
