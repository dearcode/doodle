package service

import (
	"net/http"
	"reflect"

	"dearcode.net/crab/http/server"
	"dearcode.net/doodle/util/uuid"
	"github.com/hokaccha/go-prettyjson"
)

//transport 转http请求为函数调用.
func transport(w http.ResponseWriter, r *http.Request, m reflect.Method) {
	reqType := m.Type.In(1)
	respType := m.Type.In(2).Elem()

	reqVal := reflect.New(reqType)
	respVal := reflect.New(respType)

	header := reqVal.Elem().FieldByName("RequestHeader")
	if header.IsValid() {
		s := r.Header.Get("Session")
		if s == "" {
			s = uuid.String()
		}
		header.FieldByName("Session").SetString(s)
		header.FieldByName("Request").Set(reflect.ValueOf(*r))
	}

	//先解析url中参数
	if err := server.ParseVars(r, reqVal.Interface()); err != nil {
		server.SendErrorDetail(w, http.StatusBadRequest, nil, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet, http.MethodDelete, http.MethodPost, http.MethodPut:
	default:
		server.SendResponse(w, http.StatusBadRequest, "unspport method %v", r.Method)
		return
	}

	argv := []reflect.Value{reflect.New(m.Type.In(0)).Elem(), reqVal.Elem(), respVal}
	m.Func.Call(argv)

	if _, ok := r.URL.Query()["_v"]; ok {
		b, _ := prettyjson.Marshal(respVal.Interface())
		w.Write(b)
		w.Write([]byte("\n"))
		return
	}

	server.SendData(w, respVal.Interface())
}

func (s *Service) handler(w http.ResponseWriter, r *http.Request) {
	m, ok := s.router.get(r.Method, r.URL.Path)

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	transport(w, r, m)
}
