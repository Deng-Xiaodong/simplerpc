package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"reflect"
	"simplerpc/codec"
	"strings"
	"sync"
)

type Server struct {
	services sync.Map
}

func newServer() *Server {
	return &Server{}
}

var defaultServer = newServer()

func (s *Server) Accept(lis net.Listener) {

	for {
		conn, err := lis.Accept()
		if err != nil {
			continue
		}
		go s.serverConn(conn)
	}
}

func Accept(lis net.Listener) {
	defaultServer.Accept(lis)
}

func (s *Server) serverConn(conn net.Conn) {
	//解析json格式的协议探头
	//检验魔数，判断是否需要继续解析
	//拿到编码方式，新建编解码对象，并通过它去执行下一步

	option := new(codec.Option)
	if err := json.NewDecoder(conn).Decode(option); err != nil {
		log.Fatalf("server error: parse option err: %v", err)
	}
	if option.MagicNumber != codec.MagicNum {
		log.Fatalf("server error: invalid magic number:%v", option.MagicNumber)
	}
	f, ok := codec.CodecFuncTable[option.CodecType]
	if !ok {
		log.Fatalf("server error: invailid codec type:%v", option.CodecType)
	}
	s.serverCodec(f(conn))
}

type request struct {
	h          *codec.Header
	argv, rlyv reflect.Value
	sviv       *Service
	mType      *MethodType
}

var invalidRequest = struct{}{}

func (s *Server) serverCodec(cc codec.Codec) {
	//解析header和body，执行方法，写回结果
	// readRequest、handleRequest、sendResponse
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		req, err := s.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Err = err.Error()
			s.sendResponse(cc, req.h, invalidRequest, sending, wg)
			continue
		}
		wg.Add(1)
		go s.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()

	//需要关闭资源的地方
	_ = cc.Close()

}

func (s *Server) readRequest(cc codec.Codec) (*request, error) {
	//先解析请求头
	var h = new(codec.Header)
	if err := cc.ReadHeader(h); err != nil {
		log.Fatalf("rpc server: read header err:%v", err)
	}
	//解析请求体
	req := &request{h: h}
	service, mType, err := s.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.sviv = service
	req.mType = mType

	req.argv = mType.newArgv()
	req.rlyv = mType.newRlyv()
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	if err := cc.ReadBody(argvi); err != nil {
		log.Fatalf("rpc server: read body err:%v", err)
	}

	return req, nil
}
func (s *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex, wg *sync.WaitGroup) {

	sending.Lock()
	defer sending.Unlock()
	if err := cc.WriteHeader(h); err != nil {
		log.Println(err)
		return
	}
	if err := cc.WriteBody(body); err != nil {
		log.Println(err)
		return
	}
}

func (s *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {

	defer wg.Done()
	err := req.sviv.call(req.mType, req.argv, req.rlyv)
	if err != nil {
		log.Fatalf("rpc server:mehode:%s fail calling", req.mType.method.Name)
	}
	s.sendResponse(cc, req.h, req.rlyv.Interface(), sending, wg)
}

func (s *Server) Registry(src interface{}) error {
	name := reflect.Indirect(reflect.ValueOf(src)).Type().Name()
	_, exist := s.services.Load(name)
	if exist {
		return errors.New(fmt.Sprintf("rpc server:service:%s has existed", name))
	}

	service := newService(src)
	s.services.Store(service.name, service)
	return nil
}
func Registry(src interface{}) error {
	return defaultServer.Registry(src)
}

func (s *Server) findService(serviceName string) (service *Service, mType *MethodType, err error) {
	split := strings.Split(serviceName, ".")

	svi, ok := s.services.Load(split[0])
	if !ok {
		err = errors.New(fmt.Sprintf("rpc server:service:%s not exist", split[0]))
		return
	}
	service = svi.(*Service)

	mType = service.methods[split[1]]
	if mType == nil {
		err = errors.New(fmt.Sprintf("rpc server:invalid method:%s", split[1]))
	}

	return
}
