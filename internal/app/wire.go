//go:build wireinject
// +build wireinject

package app

// This file is a pre-configured Google Wire backup boilerplate.
// To use Google Wire automated dependency injection instead of Manual DI:
//
// 1. Install Google Wire CLI:
//    go install github.com/google/wire/cmd/wire@latest
//
// 2. Open this directory in your terminal and run:
//    wire
//
// 3. This will generate a 'wire_gen.go' file which wires all interfaces automatically.
//
// 4. Update cmd/server/main.go to call app.InitializeApp(cfg, log) instead of app.NewApp(cfg, log).

import (
	"github.com/google/wire"
	productRepository "github.com/tranvux/boilerplate_golang/internal/modules/product/repository"
	productUsecase "github.com/tranvux/boilerplate_golang/internal/modules/product/usecase"
	"github.com/tranvux/boilerplate_golang/pkg/database"
	"github.com/tranvux/boilerplate_golang/pkg/messaging"
)

// ProductModuleSet groups all providers for the Product modular package
var ProductModuleSet = wire.NewSet(
	productRepository.NewPostgresProductRepository,
	productUsecase.NewProductUsecase,
)

// InfrastructureSet groups database and event brokers
var InfrastructureSet = wire.NewSet(
	database.NewPostgresDB,
	database.NewRedisClient,
	messaging.NewKafkaProducer,
)

// InitializeApp resolves dependencies automatically and builds the App struct.
func InitializeApp(cfg *config.Config, log logger.Logger) (*App, error) {
	wire.Build(
		InfrastructureSet,
		ProductModuleSet,
		// We use wire.Struct to fill all fields in the App struct automatically
		// based on the resolved providers above.
		wire.Struct(new(App), "Cfg", "Log", "Postgres", "Redis", "Producer"),
	)
	return nil, nil
}
