package proxy

import (
	"fmt"
	"sync/atomic"

	"github.com/ismailcanuslu/ayws-gateway/config"
)

// Balancer, upstream adresler arasında round-robin dağıtım yapar.
type Balancer struct {
	upstreams []string
	counter   atomic.Uint64
}

// NewBalancer verilen route listesinden upstream havuzu oluşturur.
// İleride her route için birden fazla upstream desteklemek kolaylaşır.
func NewBalancer(routes []config.RouteConfig) *Balancer {
	ups := make([]string, 0, len(routes))
	for _, r := range routes {
		ups = append(ups, r.Upstream)
	}
	return &Balancer{upstreams: ups}
}

// Next, bir sonraki upstream adresini döner.
func (b *Balancer) Next() (string, error) {
	if len(b.upstreams) == 0 {
		return "", fmt.Errorf("upstream listesi boş")
	}
	idx := b.counter.Add(1) - 1
	return b.upstreams[idx%uint64(len(b.upstreams))], nil
}
