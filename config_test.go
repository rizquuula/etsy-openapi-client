package etsy_test

import (
	"context"
	"testing"

	etsy "github.com/rizquuula/etsy-openapi-client"
)

func TestNewConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  etsy.Config
	}{
		{name: "empty config"},
		{name: "missing shared secret", cfg: etsy.Config{Keystring: "key"}},
		{
			name: "both static token and source",
			cfg: etsy.Config{
				Keystring:    "key",
				SharedSecret: "secret",
				AccessToken:  "token",
				TokenSource:  tokenSourceFunc(func(context.Context) (string, error) { return "token", nil }),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := etsy.New(tt.cfg); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
