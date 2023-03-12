package registry

import (
	"errors"
	"fmt"
	"log"
	"net"
	"simplerpc/client"
	"simplerpc/discovery"
	"simplerpc/server"
	"sync"
	"testing"
	"time"
)

type Foo int
type Args struct{ A, B int }

func (f *Foo) sum(args Args, rly *int) error {
	*rly = args.A + args.B
	return nil
}
func (f *Foo) Sum(args map[int]int, rly *map[int]int) error {

	var ok bool
	if _, ok = args[0]; !ok {
		return errors.New("no value")
	}
	if _, ok = args[1]; !ok {
		return errors.New("no value")
	}

	(*rly)[0] = args[0] + args[1]
	return nil
}
func (f *Foo) Echo(s string, rly *string) error {
	*rly = s
	return nil
}
func startServer(registry string) {

	_ = server.Registry(new(Foo))
	lis, _ := net.Listen("tcp", ":0")

	go Heartbeat(registry, lis.Addr().String(), 0)
	server.Accept(lis)

}
func StartRegistry(registry string) {
	lis, err := net.Listen("tcp", registry)
	if err != nil {
		log.Fatal(err)
	}
	StartRegistryServer(lis)
}
func TestStartRegistryServer(t *testing.T) {
	registry_http := "http://localhost:9090"
	registry_tcp := "localhost:9090"
	var wg sync.WaitGroup

	go StartRegistry(registry_tcp)
	go startServer(registry_http)
	go startServer(registry_http)

	time.Sleep(5 * time.Second)

	d := discovery.NewServerDiscovery(registry_http, time.Minute)
	client, err := client.Dial("tcp", d)
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()

			//args := Args{A: num, B: num * 2}
			args := map[int]int{0: num + 1, 1: 2 * num}
			rly := new(map[int]int)
			if err := client.Call("Foo.Sum", args, rly); err != nil {
				t.Error(err)
			}
			fmt.Printf("%v+%v=%v\n", args[0], args[1], *rly)
		}(i)
	}
	wg.Wait()

}
