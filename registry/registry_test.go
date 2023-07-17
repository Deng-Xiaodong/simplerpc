package registry

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"simplerpc/client"
	"simplerpc/codec"
	"simplerpc/discovery"
	"simplerpc/grpc/demo/service"
	"simplerpc/server"
	"sync"
	"testing"
	"time"
)

type FDD int

func (f *FDD) Shell(args service.Foo, rly *service.Foo) error {
	//m := new(service.Foo)
	//args.Data.UnmarshalTo(m)
	rly.Name = "foo:" + args.Name
	return nil
}

//func (f *FDD) Sum(args Args, rly *int) error {
//
//	*rly = args.A + args.B
//	return nil
//}

func startServer(registry string) {

	_ = server.Registry(new(FDD))
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
	client, err := client.Dial("tcp", d, &codec.Option{
		CodecType: codec.ProtoType, MagicNumber: codec.MagicNum})
	//client, err := client.Dial("tcp", d)
	if err != nil {
		t.Error(err)
	}
	s := []string{"a", "b", "c", "d", "e", "f", "g"}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			args := &service.Foo{Name: s[rand.Intn(7)]}
			rly := new(service.Foo)
			if err := client.Call("FDD.Shell", args, rly); err != nil {
				t.Error(err)
			}
			fmt.Printf("%v-->%v\n", args.Name, rly.Name)
		}()
	}
	wg.Wait()

}
