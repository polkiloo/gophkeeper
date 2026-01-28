package server_test

import (
	"testing"

	"go.uber.org/fx"

	"gophkeeper/internal/serverapp"
)

func TestFxGraphValid(t *testing.T) {
	app := fx.New(
		serverapp.AppModule,
	)
	if err := app.Err(); err != nil {
		t.Fatalf("fx graph invalid: %v", err)
	}
}
