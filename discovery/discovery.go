package discovery

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Discovery interface {
	Refresh() error // refresh from remote registry
	Update(servers []string)
	Get() (string, error)
	GetAll() ([]string, error)
}
type ServerDiscovery struct {
	servers  []string
	mu       sync.Mutex
	index    uint64
	registry string
	timeout  time.Duration
	lastTime time.Time
}

func NewServerDiscovery(registry string, timeout time.Duration) *ServerDiscovery {

	d := &ServerDiscovery{
		servers:  make([]string, 0),
		registry: registry,
		timeout:  timeout,
	}
	//err := d.Refresh()
	//if err != nil {
	//	log.Fatalf("rpc discovory: newdiscovery error when first refresh%v", err)
	//}
	return d

}

func (d *ServerDiscovery) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lastTime.Add(d.timeout).After(time.Now()) {
		return nil
	}
	log.Printf("rpc discovery: refresh servers from registry:%s", d.registry)
	resp, err := http.Get(d.registry)
	if err != nil {
		return err
	}
	servers := strings.Split(resp.Header.Get("Rpc"), ",")
	d.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			d.servers = append(d.servers, strings.TrimSpace(server))
		}
	}
	d.lastTime = time.Now()
	return nil
}

func (d *ServerDiscovery) Update(servers []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	d.lastTime = time.Now()

}

func (d *ServerDiscovery) Get() (string, error) {
	if err := d.Refresh(); err != nil {
		return "", err
	}
	index := d.index % uint64(len(d.servers))
	atomic.AddUint64(&d.index, 1)
	return d.servers[index], nil
}

func (d *ServerDiscovery) GetAll() ([]string, error) {
	if err := d.Refresh(); err != nil {
		return []string{}, err
	}
	return d.servers, nil
}
