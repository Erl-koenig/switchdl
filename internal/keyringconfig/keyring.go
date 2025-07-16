package keyringconfig

import (
	"fmt"
	"log"

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

	log.Printf("Keyring error: %v", err)
	if err != keyring.ErrNotFound {
		return "", fmt.Errorf("failed to access keyring: %w. Ensure the keyring service is running and you have appropriate permissions", err)
	}

	return "", fmt.Errorf("access token not found in keyring for service '%s' and user '%s'. Run 'switchdl configure' or provide it with the --token flag or SWITCHDL_TOKEN environment variable", Service, User)
}
