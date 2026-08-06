package main

// Example code
// https://www.speakeasy.com/blog/building-speakeasy-openapi-go-library
// https://github.com/speakeasy-api/openapi/

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"uuid"

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

func MapErrorToHTTP(err error) (int, string) {
	var conflict *core.ConflictError
	var notFound *core.NotFoundError
	var domainErr *core.DomainError

	switch {
	case errors.As(err, &conflict):
		return http.StatusConflict, conflict.Error()

	case errors.As(err, &notFound):
		return http.StatusNotFound, notFound.Error()

	case errors.As(err, &domainErr):
		// Generic domain rule violation fallback
		return http.StatusUnprocessableEntity, domainErr.Error()

	default:
		// Internal server errors / infrastructure issues should never leak details
		return http.StatusInternalServerError, "an unexpected error occurred"
	}
}
