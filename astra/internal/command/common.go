package command

import (
	"context"
	"fmt"
	"net/http"

	"github.com/datastax/astra-db-go/astra/internal/constants"
	"github.com/datastax/astra-db-go/astra/options"
)

func resolveToken(ctx context.Context, provider options.TokenProvider) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("no token provider configured")
	}
	token, err := provider.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get token from provider: %w", err)
	}
	return token, nil
}

func setCommonHeaders(headers http.Header, callers []options.Caller) {
	headers.Set("Accept", "application/json")
	headers.Set("Content-Type", "application/json")

	userAgent := constants.LibName + "/" + constants.LibVersion
	for _, caller := range callers {
		if caller.Version != "" {
			userAgent += " " + caller.Name + "/" + caller.Version
		} else {
			userAgent += " " + caller.Name
		}
	}
	headers.Set("User-Agent", userAgent)
}
