package gormx

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"gorm.io/gorm/schema"

	"github.com/saltfishpr/pkg/crypto"
)

// ctxKeyCryptor avoids collisions with context keys from other packages.
var ctxKeyCryptor int

// WithCryptoKey stores an AES key in ctx for [SecureString]. The key must be
// 16, 24, or 32 bytes.
func WithCryptoKey(ctx context.Context, key []byte) context.Context {
	return context.WithValue(ctx, &ctxKeyCryptor, crypto.New(key))
}

// CryptorFromContext returns the cryptor stored by [WithCryptoKey], or nil.
func CryptorFromContext(ctx context.Context) *crypto.Cryptor {
	if c, ok := ctx.Value(&ctxKeyCryptor).(*crypto.Cryptor); ok {
		return c
	}
	return nil
}

// SecureString transparently encrypts on write and decrypts on read when used
// as a GORM model field.
//
// Encryption uses the [crypto.Cryptor] stored in ctx by [WithCryptoKey]. If no
// cryptor is present, values are written as plain text and read back without
// decryption.
//
// SecureString only applies to model fields. Raw SQL conditions such as
// Where("secret = ?", value) are not encrypted automatically.
type SecureString string

var _ schema.SerializerInterface = (*SecureString)(nil)

// Scan reads a database value and decrypts it when a cryptor is present in
// ctx. A nil value clears the receiver.
func (s *SecureString) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue any) error {
	if dbValue == nil {
		*s = ""
		return nil
	}

	var value string
	switch v := dbValue.(type) {
	case []byte:
		value = string(v)
	case string:
		value = v
	default:
		return errors.Errorf("gormx: scan field %s: unexpected database value type %T", field.Name, dbValue)
	}

	c := CryptorFromContext(ctx)
	if c == nil {
		*s = SecureString(value)
		return nil
	}

	decrypted, err := c.Decrypt(value)
	if err != nil {
		return errors.Wrapf(err, "gormx: decrypt field %s", field.Name)
	}

	*s = SecureString(decrypted)
	return nil
}

// Value encrypts the field value when a cryptor is present in ctx. Empty and
// nil values are written as-is.
func (*SecureString) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue any) (any, error) {
	var val SecureString
	switch v := fieldValue.(type) {
	case SecureString:
		val = v
	case *SecureString:
		if v == nil {
			return nil, nil
		}
		val = *v
	case nil:
		return nil, nil
	default:
		return nil, errors.Errorf("gormx: encode field %s: unexpected field value type %T", field.Name, fieldValue)
	}

	value := string(val)
	if value == "" {
		return "", nil
	}

	c := CryptorFromContext(ctx)
	if c == nil {
		return value, nil
	}

	encrypted, err := c.Encrypt([]byte(value))
	if err != nil {
		return nil, errors.Wrapf(err, "gormx: encrypt field %s", field.Name)
	}

	return encrypted, nil
}
