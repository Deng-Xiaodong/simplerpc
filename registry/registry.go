package registry

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Registry struct {
	kv      map[string]*serviceItem
	mu      sync.Mutex
	timeout time.Duration
}

type serviceItem struct {
	start   time.Time
	address string
}

const (
	defaultTimeout = 5 * time.Second
)

func newRegistry(timeout time.Duration) *Registry {
	return &Registry{timeout: timeout, kv: make(map[string]*serviceItem)}
}

var defaultRegistry = newRegistry(defaultTimeout)

func (r *Registry) putServer(addr string) {

	r.mu.Lock()
	defer r.mu.Unlock()
	server, ok := r.kv[addr]
	if !ok {
		r.kv[addr] = &serviceItem{address: addr, start: time.Now()}
	} else {
		server.start = time.Now()
	}
}

func (r *Registry) aliveServers() []string {

	r.mu.Lock()
	defer r.mu.Unlock()
	servers := make([]string, 0)
	for _, item := range r.kv {
		if r.timeout == 0 || item.start.Add(r.timeout).After(time.Now()) {
			servers = append(servers, item.address)
		} else {
			delete(r.kv, item.address)
		}
	}
	return servers
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case "GET":
		log.Println("http get")
		w.Header().Set("Rpc", strings.Join(r.aliveServers(), ","))
	case "POST":
		addr := req.Header.Get("Rpc")
		log.Printf("post addr:%s", addr)
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
func Heartbeat(registry, addr string, duration time.Duration) {

	if duration == 0 {
		duration = defaultTimeout - time.Duration(1)*time.Second
	}
	err := sendHeart(registry, addr)
	t := time.NewTicker(duration)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err == nil {
			<-t.C
			err = sendHeart(registry, addr)
		}
		fmt.Printf("sendHeart error:%v", err)
	}()
	wg.Wait()
}
func sendHeart(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)

	req.Header.Set("Rpc", addr)

	if resp, err := client.Do(req); err != nil || resp.StatusCode != 200 {
		log.Fatalf("sendHeart error:%s", resp.Status)
	}
	return nil
}

func StartRegistryServer(lis net.Listener) error {
	return http.Serve(lis, defaultRegistry)
}
