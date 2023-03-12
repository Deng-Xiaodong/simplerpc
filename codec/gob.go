package codec

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	enc  *gob.Encoder
	dec  *gob.Decoder
}

func NewGobCodec(conn io.ReadWriteCloser) Codec {

	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		enc:  gob.NewEncoder(buf),
		dec:  gob.NewDecoder(conn),
	}
}

func (g *GobCodec) Close() error {
	return g.conn.Close()
}

func (g *GobCodec) ReadHeader(header *Header) error {
	err := g.dec.Decode(header)
	if err != nil {
		log.Println("codec error: read header err")
	}
	return err
}

func (g *GobCodec) ReadBody(body interface{}) error {
	fmt.Printf("bodyT:%T", body)
	err := g.dec.Decode(body)
	if err != nil {
		log.Println("codec error: read body err")
	}
	return err
}

func (g *GobCodec) WriteHeader(header *Header) error {
	defer g.buf.Flush()
	err := g.enc.Encode(header)
	if err != nil {
		log.Println("codec error: write header err")
		return err
	}
	return nil

}

func (g *GobCodec) WriteBody(body interface{}) error {
	defer g.buf.Flush()
	err := g.enc.Encode(body)
	if err != nil {
		log.Println("codec error: write body err")
		return err
	}
	return nil
}
