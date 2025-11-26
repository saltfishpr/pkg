package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/saltfishpr/pkg/consisthash"
	"github.com/saltfishpr/pkg/daemon"
	"github.com/saltfishpr/pkg/microservice/discovery"
)

var ErrNoInstanceFound = errors.New("no instance found")

type ServiceResolver interface {
	// Lookup looks up an instance by the given key.
	Lookup(ctx context.Context, key string) (discovery.Instance, error)
}

type serviceResolver struct {
	daemon.BaseDaemon
	key      string // service key
	provider discovery.Provider
	logger   *slog.Logger

	replicas int
	ring     atomic.Value // *consisthash.Ring[Instance]

	shutdownCh chan struct{}

	refreshCh             chan struct{}
	minRefreshInterval    time.Duration
	tickerRefreshInterval time.Duration

	refreshMu       sync.Mutex
	lastRefreshTime time.Time
}

type ServiceResolverOption func(*serviceResolver)

func WithServiceResolverReplicas(replicas int) ServiceResolverOption {
	return func(sr *serviceResolver) {
		sr.replicas = replicas
	}
}

func NewServiceResolver(key string, provider discovery.Provider, opts ...ServiceResolverOption) *serviceResolver {
	sr := &serviceResolver{
		key:                   key,
		provider:              provider,
		replicas:              300,
		shutdownCh:            make(chan struct{}),
		refreshCh:             make(chan struct{}),
		minRefreshInterval:    5 * time.Second,
		tickerRefreshInterval: 10 * time.Second,
	}
	sr.ring.Store(newRing(sr.replicas, []discovery.Instance{}))
	for _, opt := range opts {
		opt(sr)
	}
	return sr
}

func (sr *serviceResolver) Start() error {
	if err := sr.BaseDaemon.Start(); err != nil {
		return fmt.Errorf("serviceResolver start failed: %w", err)
	}

	if err := sr.refresh(false); err != nil {
		return err
	}
	go sr.refreshLoop()
	return nil
}

func (sr *serviceResolver) Stop() error {
	if err := sr.BaseDaemon.Stop(); err != nil {
		return fmt.Errorf("serviceResolver stop failed: %w", err)
	}

	close(sr.shutdownCh)
	return nil
}

func (sr *serviceResolver) Lookup(ctx context.Context, key string) (discovery.Instance, error) {
	ring := sr.ring.Load().(*consisthash.Ring[discovery.Instance])
	instance, ok := ring.Get(key)
	if !ok {
		return nil, ErrNoInstanceFound
	}
	return instance, nil
}

func (sr *serviceResolver) refreshLoop() {
	ticker := time.NewTicker(sr.tickerRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sr.shutdownCh:
			return
		case <-sr.refreshCh:
			if err := sr.refresh(true); err != nil {
				sr.logger.Error("refresh failed", "err", err)
			}
		case <-ticker.C:
			if err := sr.refresh(true); err != nil {
				sr.logger.Error("refresh failed", "err", err)
			}
		}
	}
}

func (sr *serviceResolver) refresh(lazy bool) error {
	sr.refreshMu.Lock()
	defer sr.refreshMu.Unlock()

	if lazy && time.Since(sr.lastRefreshTime) < sr.minRefreshInterval {
		return nil
	}

	instances, err := sr.provider.Discover(context.Background(), sr.key)
	if err != nil {
		return err
	}
	ring := newRing(sr.replicas, instances)
	sr.ring.Store(ring)
	sr.lastRefreshTime = time.Now()
	return nil
}

func newRing(replicas int, instances []discovery.Instance) *consisthash.Ring[discovery.Instance] {
	ring := consisthash.NewRing(replicas, func(node discovery.Instance) string {
		return node.Identifier()
	})
	ring.Add(instances...)
	return ring
}
