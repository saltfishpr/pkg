package gormx

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"gorm.io/gorm/schema"

	"github.com/saltfishpr/pkg/crypto"
)

var ctxKeyCryptoKey int

// WithCryptoKey 向 context 中存入加密密钥。
func WithCryptoKey(ctx context.Context, key []byte) context.Context {
	return context.WithValue(ctx, &ctxKeyCryptoKey, key)
}

// GetCryptoKey 从 context 中获取加密密钥。
// 如果未设置密钥,返回 nil。
func GetCryptoKey(ctx context.Context) []byte {
	if key, ok := ctx.Value(&ctxKeyCryptoKey).([]byte); ok {
		return key
	}
	return nil
}

// SecureString 实现了 [schema.SerializerInterface],透明地加密/解密字符串值。
// 用作数据库字段类型时,写入时自动加密,读取时自动解密。
// 通过 WithCryptoKey 设置的密钥进行加解密;未设置密钥则按明文处理。
type SecureString string

var _ schema.SerializerInterface = (*SecureString)(nil)

// Scan 实现 schema.SerializerInterface 接口,从数据库读取数据并解密。
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

// Value 实现 schema.SerializerInterface 接口,加密数据并写入数据库。
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
