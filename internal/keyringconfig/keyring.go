// Package keyringconfig manages access token storage and retrieval using the system keyring
package keyringconfig

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	Service = "switchdl" // Keyring service name
	User    = "default"  // Default keyring user
)

func GetAccessToken(currentToken string) (string, error) {
	if currentToken != "" { // If already provided by flag
		return currentToken, nil
	}

	token, err := keyring.Get(Service, User)
	if err == nil {
		return token, nil
	}

	if !errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf(
			"failed to access keyring: %w. Ensure the keyring service is running and you have appropriate permissions",
			err,
		)
	}

	return "", fmt.Errorf(
		"access token not found in keyring for service '%s' and user '%s'. Run 'switchdl configure' or provide it with the --token flag or SWITCHDL_TOKEN environment variable",
		Service,
		User,
	)
}

func SetAccessToken(token string) error {
	if token == "" {
		return errors.New("access token cannot be empty")
	}
	if err := keyring.Set(Service, User, token); err != nil {
		return fmt.Errorf("failed to save token to keyring: %w", err)
	}
	return nil
}

func DeleteAccessToken() error {
	if err := keyring.Delete(Service, User); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}
