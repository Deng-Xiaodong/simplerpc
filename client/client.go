package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"simplerpc/codec"
	"simplerpc/discovery"
	"sync"
)

type Call struct {
	//header
	//所有请求可以共用一个header
	serviceMethod string
	seq           uint64
	args, rly     interface{}
	err           error
	doneChan      chan struct{}
}

func (c *Call) done() {
	c.doneChan <- struct{}{}
}

type Client struct {

	//基本要素
	//1 维护用户连接的编解码器
	//2 维护共用的请求头和当前调用序列号
	cc     codec.Codec
	header *codec.Header
	seq    uint64

	//调用列表以及作为临界资源，并发访问时需要的锁
	pending     map[uint64]*Call
	sending, mu sync.Mutex

	//记录用户连接状态
	//1 closed 用户主动关闭连接
	//2 shutdown 系统发生故障
	closed   bool
	shutdown bool
}

func (c *Client) Close() error {
	return c.cc.Close()
}

// 在注册新的call前需要判断客户端是否处于可用状态
func (c *Client) isUnavailable() bool {
	return c.closed || c.shutdown
}

func (c *Client) registryCall(call *Call) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isUnavailable() {
		return 0, errors.New("rpc client: error shutdown")
	}
	call.seq = c.seq
	c.seq++
	c.pending[call.seq] = call
	return call.seq, nil
}

func (c *Client) removeCall(seq uint64) *Call {
	c.mu.Lock()
	defer c.mu.Unlock()
	call := c.pending[seq]
	delete(c.pending, seq)
	return call
}

func (c *Client) terminalAllCalls() {
	c.sending.Lock()
	defer c.sending.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	c.shutdown = true
	for _, call := range c.pending {
		call.done()
	}
}

func Dial(network string, d discovery.Discovery, options ...*codec.Option) (*Client, error) {
	option, err := parseOption(options...)
	if err != nil {
		return nil, err
	}
	rpcAddr, disErr := d.Get()
	if disErr != nil {
		return nil, disErr
	}
	conn, dialErr := net.Dial(network, rpcAddr)
	if dialErr != nil {
		return nil, dialErr
	}
	if err := json.NewEncoder(conn).Encode(option); err != nil {
		return nil, err
	}
	f := codec.CodecFuncTable[option.CodecType]
	return newClient(f(conn)), nil
}

func (c *Client) Call(serviceMethod string, args interface{}, rly interface{}) error {

	call := &Call{
		serviceMethod: serviceMethod,
		args:          args,
		rly:           rly,
		doneChan:      make(chan struct{}, 1),
	}
	c.send(call)
	<-call.doneChan
	return call.err

}
func newClient(cc codec.Codec) *Client {

	client := &Client{
		header:  new(codec.Header),
		cc:      cc,
		seq:     1,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}
func (c *Client) receive() {

	var err error
	for err == nil {
		//解析头部
		h := new(codec.Header)
		if err = c.cc.ReadHeader(h); err != nil {
			break
		}
		// 通过seq拿到回应的call
		call := c.removeCall(h.Seq)
		//call有三种可能状态
		//1 为nil：发送请求未完整或者被取消了，但服务器处理了
		//2 不为nil，但是Err不为nil：处理出错了
		//3 正常处理
		switch {
		case call == nil:
			err = errors.New("rpc client: handled not existed call")
		case h.Err != "":
			call.err = errors.New(h.Err)
			err = call.err
			call.done()
		default:
			err = c.cc.ReadBody(call.rly)
			call.done()
		}

	}
	c.terminalAllCalls()
}
func parseOption(options ...*codec.Option) (*codec.Option, error) {
	if len(options) == 0 {
		return codec.DefaultOption, nil
	}
	if len(options) > 1 {
		return nil, errors.New("rpc client: the length of options must be  one or empty")
	}
	return options[0], nil
}

func (c *Client) send(call *Call) {

	c.sending.Lock()
	defer c.sending.Unlock()
	seq, err := c.registryCall(call)
	if err != nil {
		call.err = err
		call.done()
		return
	}

	c.header.ServiceMethod = call.serviceMethod
	c.header.Seq = seq
	c.header.Err = ""
	if c.cc.WriteHeader(c.header) != nil || c.cc.WriteBody(call.args) != nil {
		c.removeCall(seq)
		call.err = fmt.Errorf("rpc client:call_%d failed to write request", call.seq)
		call.done()
	}

}
