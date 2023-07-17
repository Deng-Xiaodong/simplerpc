package codec

import (
	"io"
	"simplerpc/grpc/demo/service"
)

// 协议部分
// 请求探头：魔数、编码方式
// 请求头：服务方法、请求序号、请求错误信息
// 请求体：方法参数、方法结果

//

const (
	MagicNum       = 0x123af
	GobType   Type = "application/gob"
	JsonType  Type = "application/json"
	ProtoType Type = "application/proto"
)

type Type = string
type NewCodecFunc func(closer io.ReadWriteCloser) Codec

type Option struct {
	MagicNumber int64
	CodecType   Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNum,
	CodecType:   GobType,
}

//	type Header struct {
//		ServiceMethod string
//		Seq           uint64
//		Err           string
//	}
type Header = service.Header
type Body = service.Body
type Codec interface {
	io.Closer
	ReadHeader(header *Header) error
	ReadBody(body interface{}) error
	//ReadBody(body *Body) error
	WriteHeader(header *Header) error
	WriteBody(body interface{}) error
	//WriteBody(body *Body) error
}

var CodecFuncTable map[Type]NewCodecFunc

func init() {
	CodecFuncTable = make(map[Type]NewCodecFunc)
	CodecFuncTable[GobType] = NewGobCodec
	CodecFuncTable[ProtoType] = NewProtoCodec
}
