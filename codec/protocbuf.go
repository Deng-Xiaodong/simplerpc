package codec

import (
	"bufio"
	"encoding/binary"
	"google.golang.org/protobuf/proto"
	"io"
)

type ProtoCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.ReadWriter
}

func NewProtoCodec(conn io.ReadWriteCloser) Codec {
	return &ProtoCodec{
		conn: conn,
		buf:  bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

func (c *ProtoCodec) Close() error {
	return c.conn.Close()
}

func (c *ProtoCodec) ReadHeader(header *Header) error {
	//将缓存的字节数组反序列化成message对象
	byte_len := make([]byte, 8)
	c.buf.Read(byte_len)
	l, _ := binary.Uvarint(byte_len)
	bytes := make([]byte, l)
	c.buf.Read(bytes)
	proto.Unmarshal(bytes, header)
	return nil
}

func (c *ProtoCodec) ReadBody(body any) error {
	//fmt.Printf("bodyT:%T", body)
	byte_len := make([]byte, 8)
	c.buf.Read(byte_len)
	l, _ := binary.Uvarint(byte_len)
	bytes := make([]byte, l)
	c.buf.Read(bytes)
	//&Body{Data: anypb.New(reflect.ValueOf(body))}
	proto.Unmarshal(bytes, body.(proto.Message))
	return nil
}

func (c *ProtoCodec) WriteHeader(header *Header) error {
	//将message对象序列化到缓存中
	byte_date, err := proto.Marshal(header)
	if err != nil {
		return err
	}
	byte_len := make([]byte, 8)
	binary.PutUvarint(byte_len, uint64(len(byte_date)))
	c.buf.Write(byte_len)
	c.buf.Write(byte_date)
	c.buf.Flush()
	return nil
}

func (c *ProtoCodec) WriteBody(body any) error {
	//将message对象序列化到缓存中
	byte_date, err := proto.Marshal(body.(proto.Message))
	if err != nil {
		return err
	}
	byte_len := make([]byte, 8)
	binary.PutUvarint(byte_len, uint64(len(byte_date)))
	c.buf.Write(byte_len)
	c.buf.Write(byte_date)
	c.buf.Flush()
	return nil
}

//func (c *ProtoCodec) Read(r *service.AddRequest) error {
//	//将缓存的字节数组反序列化成message对象
//	byte_len := make([]byte, 8)
//	c.buf.Read(byte_len)
//	l, _ := binary.Uvarint(byte_len)
//	bytes := make([]byte, l)
//	c.buf.Read(bytes)
//	proto.Unmarshal(bytes, r)
//	return nil
//}
//
//func (c *ProtoCodec) Write(r *service.AddRequest) error {
//	//将message对象序列化到缓存中
//	byte_date, err := proto.Marshal(r)
//	if err != nil {
//		return err
//	}
//	byte_len := make([]byte, 8)
//	binary.PutUvarint(byte_len, uint64(len(byte_date)))
//	c.buf.Write(byte_len)
//	c.buf.Write(byte_date)
//	c.buf.Flush()
//	return nil
//}
