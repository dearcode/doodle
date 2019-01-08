package service

import (
	"reflect"
	"sync"
)

type router struct {
	methods map[string]reflect.Method
	sync.Mutex
}

func newRouter() router {
	return router{methods: make(map[string]reflect.Method)}
}

func (r *router) add(method, path string, m reflect.Method) {
	r.Lock()
	defer r.Unlock()
	r.methods[method+path] = m
}

func (r *router) get(method, path string) (m reflect.Method, ok bool) {
	r.Lock()
	defer r.Unlock()
	m, ok = r.methods[method+path]
	return
}
