package oidcprovider

import (
	"context"
	"crypto/subtle"
	"fmt"
)

type PlainSecretHasher struct{}

func (PlainSecretHasher) Hash(_ context.Context, data []byte) ([]byte, error) {
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

func (PlainSecretHasher) Compare(_ context.Context, expected, actual []byte) error {
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return fmt.Errorf("client secret mismatch")
	}
	return nil
}
