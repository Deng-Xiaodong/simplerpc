package codec

import (
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
)

func TestGobCodec(t *testing.T) {

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {

		defer wg.Done()
		lis, err := net.Listen("tcp", ":9090")
		if err != nil {
			log.Fatal(err)
		}
		if conn, err := lis.Accept(); err == nil {

			cc := NewGobCodec(conn)
			h := new(Header)
			cc.ReadHeader(h)
			fmt.Printf("service_name:%v\n", h)
		}
	}()

	conn, err := net.Dial("tcp", ":9090")
	if err != nil {
		log.Fatal(err)
	}
	cc := NewGobCodec(conn)
	if err := cc.WriteHeader(&Header{
		ServiceMethod: "Foo.Echo",
		Seq:           1,
		Err:           "",
	}); err != nil {
		log.Println(err)
	}
	wg.Wait()
}
