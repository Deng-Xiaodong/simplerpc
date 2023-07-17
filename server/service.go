package server

import (
	"go/ast"
	"log"
	"reflect"
)

type MethodType struct {
	method           reflect.Method //call的时候需要
	argType, rlyType reflect.Type   //知道参数类型才能赋值，并传入执行call
}

func (mt *MethodType) newArgv() reflect.Value {
	var argv reflect.Value
	if mt.argType.Kind() == reflect.Ptr {
		argv = reflect.New(mt.argType.Elem())
	} else {
		argv = reflect.New(mt.argType).Elem()
	}
	return argv
}
func (mt *MethodType) newRlyv() reflect.Value {
	//var rlyv reflect.Value
	//rlyt := mt.rlyType.Elem()
	//switch rlyt.Kind() {
	//case reflect.Map:
	//	rlyv = reflect.MakeMap(rlyt).Addr()
	//case reflect.Slice:
	//	rlyv = reflect.MakeSlice(rlyt, 0, 0).Addr()
	//default:
	//	rlyv = reflect.New(rlyt)
	//}
	//return rlyv
	rlyv := reflect.New(mt.rlyType.Elem())
	switch rlyv.Elem().Kind() {
	case reflect.Map:
		rlyv.Elem().Set(reflect.MakeMap(rlyv.Elem().Type()))
	case reflect.Slice:
		rlyv.Elem().Set(reflect.MakeSlice(rlyv.Elem().Type(), 0, 0))
	}
	return rlyv

}

type Service struct {
	name    string                 //server通过该name查找到对应的service
	svit    reflect.Type           // 注册该service的method需要通过它查看所有方法并注册
	sviv    reflect.Value          //call的时候需要
	methods map[string]*MethodType //注册的结果，key为methodName，value为该method的methodType
}

func newService(src interface{}) *Service {
	s := new(Service)
	s.svit = reflect.TypeOf(src)
	s.sviv = reflect.ValueOf(src)
	//指针类型取不到正确的名字
	s.name = reflect.Indirect(s.sviv).Type().Name()
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not A valid service name", s.name)
	}
	s.registerMethods()
	return s
}

func (s *Service) registerMethods() {

	s.methods = make(map[string]*MethodType)
	for i := 0; i < s.svit.NumMethod(); i++ {

		method := s.svit.Method(i)
		if method.Type.NumIn() != 3 || method.Type.NumOut() != 1 {
			continue
		}
		if method.Type.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, rlyType := method.Type.In(1), method.Type.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(rlyType) {
			continue
		}
		s.methods[method.Name] = &MethodType{
			method:  method,
			argType: argType,
			rlyType: rlyType,
		}
		log.Printf("rpc server: %s.%s was registed successfully", s.name, method.Name)
	}

}

func (s *Service) call(m *MethodType, argv, rlyv reflect.Value) error {
	f := m.method.Func
	callRes := f.Call([]reflect.Value{s.sviv, argv, rlyv})
	if err := callRes[0].Interface(); err != nil {
		return err.(error)
	}
	return nil
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}
