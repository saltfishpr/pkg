package gormx

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"gorm.io/gorm/schema"

	"github.com/saltfishpr/pkg/crypto"
)

var ctxKeyCryptoKey int

// WithCryptoKey sets a cryptographic key in the context.
func WithCryptoKey(ctx context.Context, key []byte) context.Context {
	return context.WithValue(ctx, &ctxKeyCryptoKey, key)
}

func GetCryptoKey(ctx context.Context) []byte {
	if key, ok := ctx.Value(&ctxKeyCryptoKey).([]byte); ok {
		return key
	}
	return nil
}

// SecureString is a gorm serializer that encrypts/decrypts string values transparently
type SecureString string

var _ schema.SerializerInterface = (*SecureString)(nil)

// Scan implements the serializer interface for reading from the database
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

// Value implements the serializer interface for writing to the database
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
