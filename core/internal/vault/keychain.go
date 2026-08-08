package vault

import (
	"github.com/zalando/go-keyring"
)

const serviceName = "DevTerm"

// keychainVault stores secrets in the OS keychain.
type keychainVault struct{}

func newKeychainVault() (*keychainVault, error) {
	// Probe: try a set/get/delete cycle to verify keychain is available
	testKey := "__devterm_probe__"
	testVal := "probe"
	if err := keyring.Set(serviceName, testKey, testVal); err != nil {
		return nil, err
	}
	if _, err := keyring.Get(serviceName, testKey); err != nil {
		return nil, err
	}
	_ = keyring.Delete(serviceName, testKey)
	return &keychainVault{}, nil
}

func (v *keychainVault) Put(ref string, secret []byte) error {
	return keyring.Set(serviceName, ref, string(secret))
}

func (v *keychainVault) Get(ref string) ([]byte, error) {
	s, err := keyring.Get(serviceName, ref)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

func (v *keychainVault) Delete(ref string) error {
	return keyring.Delete(serviceName, ref)
}
