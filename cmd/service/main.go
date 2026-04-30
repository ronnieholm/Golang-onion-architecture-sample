package main

// Example code
// https://www.speakeasy.com/blog/building-speakeasy-openapi-go-library
// https://github.com/speakeasy-api/openapi/

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/ronnieholm/resellerloyalty/internal/build"
	"github.com/ronnieholm/resellerloyalty/internal/core"
	"github.com/ronnieholm/resellerloyalty/internal/infrastructure"
)

func main() {
	var (
		version   = build.Version
		buildTime = build.BuildTime
	)
	if build.Version == "" {
		version = "N/A"
	}
	if build.BuildTime == "" {
		buildTime = "N/A"
	}
	fmt.Printf("Version %s build at %s.\n\n", version, buildTime)

	config, err := infrastructure.LoadConfig("./configs/service.json")
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	ctx := context.Background()
	dispatcher := infrastructure.NewDispatcher(ctx, config)
	cmd := core.CreateCurrencyCommand{
		ID:   uuid.MustParse("282490d5-4ca1-4b07-9955-6c8bbbc4ea00"),
		Code: "USD",
	}
	_, err = dispatcher.CreateCurrency(ctx, cmd)
	if err != nil {
		log.Fatalf("Create currency: %s", err)
	}
}
