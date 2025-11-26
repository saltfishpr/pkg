// Package discovery defines interfaces for service discovery providers and instances.
package discovery

import "context"

//go:generate mockgen -typed -package mock_$GOPACKAGE -source=$GOFILE -destination=mock_$GOPACKAGE/$GOFILE

type Instance interface {
	Identifier() string
}

type Provider interface {
	Discover(ctx context.Context, key string) ([]Instance, error)
}
