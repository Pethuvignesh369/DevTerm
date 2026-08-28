//go:build !windows

package vault

func vaultKeyNeedsProtection() bool { return false }

func protectVaultKey(key []byte) ([]byte, error) {
	return append([]byte(nil), key...), nil
}

func unprotectVaultKey(protected []byte) ([]byte, error) {
	return append([]byte(nil), protected...), nil
}
