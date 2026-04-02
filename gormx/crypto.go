package gormx

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"gorm.io/gorm/schema"

	"github.com/saltfishpr/pkg/crypto"
)

// ctxKeyCryptoKey is a private, pointer-addressed context key for the AES
// encryption key, avoiding string-based key collisions.
var ctxKeyCryptoKey int

// WithCryptoKey stores an AES encryption key in ctx for use by [SecureString]
// during GORM Scan/Value operations.
func WithCryptoKey(ctx context.Context, key []byte) context.Context {
	return context.WithValue(ctx, &ctxKeyCryptoKey, key)
}

// GetCryptoKey retrieves the AES key from ctx, or nil if none is set.
func GetCryptoKey(ctx context.Context) []byte {
	if key, ok := ctx.Value(&ctxKeyCryptoKey).([]byte); ok {
		return key
	}
	return nil
}

// SecureString is a string type that transparently encrypts on write and
// decrypts on read when used as a GORM model field via the serializer
// interface ([schema.SerializerInterface]).
//
// Encryption uses AES-GCM through the [crypto.Cryptor] obtained from the
// context key set by [WithCryptoKey]. If no key is present, the value is
// stored and retrieved as plain text.
type SecureString string

var _ schema.SerializerInterface = (*SecureString)(nil)

// Scan implements [schema.SerializerInterface]. It reads a database value
// and decrypts it if a crypto key is present in ctx.
func (s *SecureString) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	if dbValue == nil {
		return nil
	}

	var value string
	switch v := dbValue.(type) {
	case []byte:
		value = string(v)
	case string:
		value = v
	default:
		return errors.Errorf("failed to unmarshal SecureString value: %#v", dbValue)
	}

	key := GetCryptoKey(ctx)
	if key == nil {
		*s = SecureString(value)
		return nil
	}

	c := crypto.New(key)
	decrypted, err := c.Decrypt(value)
	if err != nil {
		return errors.WithStack(err)
	}

	*s = SecureString(decrypted)
	return nil
}

// Value implements [schema.SerializerInterface]. It encrypts the field value
// before writing to the database if a crypto key is present in ctx.
func (*SecureString) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	val, ok := fieldValue.(SecureString)
	if !ok {
		if p, ok := fieldValue.(*SecureString); ok && p != nil {
			val = *p
		} else {
			return nil, nil
		}
	}

	value := string(val)
	if value == "" {
		return "", nil
	}

	key := GetCryptoKey(ctx)
	if key == nil {
		return value, nil
	}

	c := crypto.New(key)
	encrypted, err := c.Encrypt([]byte(value))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return encrypted, nil
}
