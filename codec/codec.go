package codec

import "io"

// 协议部分
// 请求探头：魔数、编码方式
// 请求头：服务方法、请求序号、请求错误信息
// 请求体：方法参数、方法结果

//

const (
	MagicNum      = 0x123af
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
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

type Header struct {
	ServiceMethod string
	Seq           uint64
	Err           string
}

type Codec interface {
	io.Closer
	ReadHeader(header *Header) error
	ReadBody(body interface{}) error
	WriteHeader(header *Header) error
	WriteBody(body interface{}) error
}

var CodecFuncTable map[Type]NewCodecFunc

func init() {
	CodecFuncTable = make(map[Type]NewCodecFunc)
	CodecFuncTable[GobType] = NewGobCodec
}
