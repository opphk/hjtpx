package servicediscovery

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc/resolver"
)

type consulResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	client     *ConsulService
	service    string
	instances  []*ServiceInstance
	mu         sync.RWMutex
	watchCh    chan struct{}
	cancel     context.CancelFunc
}

func NewConsulResolver(target resolver.Target, client *ConsulService) resolver.Resolver {
	return &consulResolver{
		target:  target,
		client:  client,
		service: target.Endpoint(),
		watchCh: make(chan struct{}, 1),
	}
}

func (r *consulResolver) ResolveNow(options resolver.ResolveNowOptions) {
	r.mu.Lock()
	defer r.mu.Unlock()

	instances, err := r.client.Discover(r.service)
	if err != nil {
		r.cc.ReportError(fmt.Errorf("failed to resolve service %s: %w", r.service, err))
		return
	}

	if len(instances) == 0 {
		r.cc.ReportError(fmt.Errorf("no instances found for service %s", r.service))
		return
	}

	r.instances = instances
	addresses := make([]resolver.Address, 0, len(instances))
	for _, inst := range instances {
		if inst.Healthy {
			addresses = append(addresses, resolver.Address{
				Addr:       fmt.Sprintf("%s:%d", inst.Address, inst.Port),
				ServerName: inst.Name,
			})
		}
	}

	if len(addresses) > 0 {
		r.cc.UpdateState(resolver.State{Addresses: addresses})
	}
}

func (r *consulResolver) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}

type consulBuilder struct{}

func (b *consulBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	consulAddr := target.URL.Host
	if consulAddr == "" {
		consulAddr = "localhost:8500"
	}

	config := &ConsulConfig{
		Address: consulAddr,
	}

	client, err := NewConsulService(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	r := &consulResolver{
		target: target,
		cc:     cc,
		client: client,
		service: target.Endpoint(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	go r.watch(ctx)

	return r, nil
}

func (b *consulBuilder) Scheme() string {
	return "consul"
}

func (r *consulResolver) watch(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.ResolveNow(resolver.ResolveNowOptions{})
		}
	}
}

func init() {
	resolver.Register(&consulBuilder{})
}

func RandomInstance(instances []*ServiceInstance) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}
	rand.Seed(time.Now().UnixNano())
	return instances[rand.Intn(len(instances))]
}

func RoundRobinInstance(instances []*ServiceInstance, counter *int) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}
	*counter = (*counter + 1) % len(instances)
	return instances[*counter]
}