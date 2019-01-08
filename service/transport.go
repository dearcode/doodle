package service

import (
	"net/http"
	"reflect"

	"github.com/dearcode/crab/http/server"
	"github.com/dearcode/crab/log"
	"github.com/dearcode/doodle/util/uuid"
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

	switch r.Method {
	case http.MethodGet, http.MethodDelete:
		if err := server.ParseURLVars(r, reqVal.Interface()); err != nil {
			server.SendErrorDetail(w, http.StatusBadRequest, nil, err.Error())
			return
		}
	case http.MethodPost, http.MethodPut:
		if err := server.ParseJSONVars(r, reqVal.Interface()); err != nil {
			server.SendErrorDetail(w, http.StatusBadRequest, nil, err.Error())
			return
		}
	default:
		server.SendResponse(w, http.StatusBadRequest, "unspport method %v", r.Method)
		return
	}

	argv := []reflect.Value{reflect.New(m.Type.In(0)).Elem(), reqVal.Elem(), respVal}
	m.Func.Call(argv)
	server.SendData(w, respVal.Interface())
}

func (s *Service) handler(w http.ResponseWriter, r *http.Request) {
	m, ok := s.router.get(r.Method, r.URL.Path)
	log.Debugf("m:%v, ok:%v, method:%v, path:%v", m, ok, r.Method, r.URL.Path)

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	transport(w, r, m)
}
