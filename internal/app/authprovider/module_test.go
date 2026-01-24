package authprovider

import (
	"testing"

	localauth "gophkeeper/internal/app/auth"
	"gophkeeper/internal/app/auth/keycloak"
	"gophkeeper/internal/config"
)

func TestSelectAuthUseCase(t *testing.T) {
	local := &localauth.Service{}
	kc := &keycloak.Service{}

	use := selectAuthUseCase(config.Config{AuthProvider: "keycloak"}, local, kc)
	if use != kc {
		t.Fatalf("expected keycloak")
	}

	use = selectAuthUseCase(config.Config{AuthProvider: "local"}, local, kc)
	if use != local {
		t.Fatalf("expected local")
	}
}
