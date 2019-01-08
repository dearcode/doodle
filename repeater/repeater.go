package repeater

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/dearcode/crab/http/server"
	"github.com/dearcode/crab/log"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/meta"
	"github.com/dearcode/doodle/util"
	"github.com/dearcode/doodle/util/uuid"
)

func (r *repeater) delValue(req *http.Request, v *meta.Variable) {
	log.Debugf("del %v %v", v.Postion, v.Name)
	switch v.Postion {
	case server.URI:
		m := req.URL.Query()
		m.Del(v.Name)
	case server.HEADER:
		req.Header.Del(v.Name)
	case server.FORM:
		req.Form.Del(v.Name)
	}
}

func (r *repeater) validateValue(v *meta.Variable, val string) (bool, error) {
	if !v.Required && val == "" {
		return true, nil
	}

	if v.Type == "int" {
		_, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			log.Infof("val:%s, ParseInt error:%s, in %s", val, err.Error(), v.Postion)
			return false, fmt.Errorf("key:%s val:%s is not number, in %s", v.Name, val, v.Postion)
		}
		return false, nil
	}

	if val == "" {
		return false, fmt.Errorf("key:%s not found in %s", v.Name, v.Postion)
	}

	return false, nil
}

// GetInterface 根据请求header获取对应接口
func (r *repeater) GetInterface(req *http.Request, id string) (app *meta.Application, iface *meta.Interface, err error) {
	token := req.Header.Get("Token")
	if token == "" {
		return nil, nil, errors.Trace(errNotFoundToken)
	}

	log.Infof("%s requset token is:%v", id, token)

	if app, err = dc.getApp(token); err != nil {
		log.Errorf("%s get app error,token is:%v", id, token)
		return nil, nil, errors.Trace(err)
	}
	log.Infof("%s app is:%v, user email is:%v", id, app.Name, app.Email)

	if iface, err = dc.getInterface(req.URL.Path); err != nil {
		log.Errorf("%s get interface error path:%v, user email is:%v", id, req.URL.Path, app.Email)
		return nil, nil, errors.Trace(err)
	}
	log.Infof("%s iface is:%v,user email is:%v", id, iface.Path, iface.Email)

	if iface.Method != server.RESTful && req.Method != iface.Method.String() {
		log.Errorf("%s url:%v, invalid method:%v, need:%v,user email is:%v", id, req.URL, req.Method, iface.Method, iface.Email)
		return nil, nil, fmt.Errorf("invalid method:%v, need:%v", req.Method, iface.Method)
	}

	//如果不需要验证权限，直接通过
	if !iface.Service.Validate {
		log.Debugf("%s project:%v validate is flase, app:%v", id, id, iface.Service, app)
		return
	}

	if err = dc.validateRelation(app.ID, iface.ID); err == nil {
		log.Debugf("%s project:%v iface:%v app:%v", id, iface, app)
		return
	}

	if errors.Cause(err) == errNotFound {
		return nil, nil, errors.Trace(errForbidden)
	}

	log.Errorf("%s validateRelation appId:%v,ifaceId:%v,app email:%v,iface email is:%v", id, app.ID, iface.ID, app.Email, iface.Email)
	return nil, nil, errors.Trace(err)
}

func (r *repeater) parseForm(req *http.Request, vars []*meta.Variable) error {
	//如果需要解析body，则要备份一份
	for _, v := range vars {
		if v.Postion == server.FORM {
			buf, err := ioutil.ReadAll(req.Body)
			if err != nil {
				return errors.Trace(err)
			}

			req.Body = ioutil.NopCloser(bytes.NewReader(buf))
			//执行这个操作会把body读空
			if err := req.ParseForm(); err != nil {
				return errors.Trace(err)
			}
			//再还回去
			req.Body = ioutil.NopCloser(bytes.NewReader(buf))
			break
		}
	}

	return nil
}

//Validate 验证输入参数，如果通过验证返回后端地址
func (r *repeater) Validate(req *http.Request, iface *meta.Interface) error {
	vars, err := dc.getVariable(iface.ID)
	if err != nil {
		return errors.Trace(err)
	}

	if err = r.parseForm(req, vars); err != nil {
		return errors.Trace(err)
	}

	for _, v := range vars {
		var val string
		switch v.Postion {
		case server.URI:
			val = req.URL.Query().Get(v.Name)
		case server.FORM:
			val = req.FormValue(v.Name)
		case server.HEADER:
			val = req.Header.Get(v.Name)
		}
		del, err := r.validateValue(v, val)
		if err != nil {
			return errors.Trace(err)
		}
		if del {
			r.delValue(req, v)
		}
	}

	return nil
}

func (r *repeater) microAPPBackendURL(iface *meta.Interface, req *http.Request) (string, error) {
	apps, err := bs.getMicroAPPs(iface.Backend)
	if err != nil {
		return "", errors.Trace(err)
	}
	idx := time.Now().UnixNano() % int64(len(apps))
	backend := fmt.Sprintf("http://%s:%d%s", apps[idx].Host, apps[idx].Port, iface.Path)
	//生成url参数
	if args := req.URL.Query().Encode(); args != "" {
		backend += "?" + args
	}

	return backend, nil
}

//backendURL 如果是faas的，随机访问后端地址， 如果是传统的走域名直接访问.
func (r *repeater) backendURL(iface *meta.Interface, req *http.Request) (string, error) {
	if iface.Service.Version == 1 {
		return r.microAPPBackendURL(iface, req)
	}

	uri := req.RequestURI
	//跳过一级目录
	if idx := strings.Index(uri[1:], "/"); idx > 0 {
		uri = uri[idx+2:]
	}

	//清理二级目录
	uri = strings.TrimPrefix(uri, iface.Path)
	if len(uri) < 2 {
		return iface.Backend, nil
	}

	if len(uri) > 2 && uri[1] == '?' {
		uri = uri[1:]
	}

	if uri[0] == '/' && iface.Backend[len(iface.Backend)-1] == '/' {
		uri = uri[1:]
	}

	return iface.Backend + uri, nil
}

//buildRequest 生成后端请求request,清理无用的请求参数
func (r *repeater) buildRequest(id string, iface *meta.Interface, req *http.Request) error {
	backend, err := r.backendURL(iface, req)
	if err != nil {
		return errors.Trace(err)
	}

	if req.URL, err = url.Parse(backend); err != nil {
		return errors.Trace(err)
	}

	req.Host = req.URL.Host
	req.Header.Set("User-Agent", "APIGate "+util.GitTime)
	req.Header.Del("Token")
	req.RequestURI = ""
	req.Header.Set("Session", id)

	return nil
}

func (r *repeater) requestBody(id string, req *http.Request) error {
	//接口接收到请求的详细信息
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return errors.Trace(err)
	}

	log.Infof("%s data:%v", id, string(buf))
	//再还回去
	req.Body = ioutil.NopCloser(bytes.NewReader(buf))

	return nil
}

func (r *repeater) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	switch errors.Cause(err) {
	case errForbidden:
		status = http.StatusForbidden
	case errInvalidPath, errInvalidToken, errNotFound:
		status = http.StatusNotFound
	case errNotFoundToken:
		status = http.StatusUnauthorized
	}

	w.WriteHeader(status)
	w.Write([]byte(err.Error()))
}

//ServeHTTP 入口
func (r *repeater) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	id := uuid.String()
	w.Header().Add("Session", id)

	defer func() {
		if e := recover(); e != nil {
			log.Errorf("%s recover %v", id, e)
			log.Errorf("%s", debug.Stack())
			r.writeError(w, fmt.Errorf("%v", e))
		}
	}()

	log.Infof("%s url:%v method:%v", id, req.URL, req.Method)

	//解析并记录请求body
	if err := r.requestBody(id, req); err != nil {
		log.Errorf("%v read body error:%v", id, errors.ErrorStack(err))
		r.writeError(w, err)
		return
	}

	//查找对应接口信息
	app, iface, err := r.GetInterface(req, id)
	if err != nil {
		log.Errorf("%s error:%s", id, errors.ErrorStack(err))
		r.writeError(w, err)
		return
	}
	log.Infof("%s app:%s email:%s, interface:%s email:%s", id, app.Name, app.Email, iface.Name, iface.Email)

	//验证输入参数
	if err = r.Validate(req, iface); err != nil {
		log.Errorf("%s validate error:%s", id, errors.ErrorStack(err))
		r.writeError(w, err)
		return
	}
	log.Infof("%s validate success", id)

	//生成后端请求
	if err = r.buildRequest(id, iface, req); err != nil {
		log.Errorf("%s build request error:%s", id, errors.ErrorStack(err))
		r.writeError(w, err)
		return
	}
	log.Infof("%s backend url:%s method:%s begin", id, req.URL, iface.Method)

	b := time.Now()
	rb, code, err := util.DoRequest(req)
	cost := time.Since(b) / time.Millisecond

	if err != nil {
		stats.failed(id, app.ID, iface.ID, err.Error())
		log.Errorf("%s used:%dms end error:%s", id, cost, err.Error())
		r.writeError(w, err)
		return
	}

	if code != http.StatusOK {
		stats.failed(id, app.ID, iface.ID, fmt.Sprintf("invalid http status:%v", code))
		log.Errorf("%s used:%dms end failed, code:%d", id, cost, code)
	} else {
		stats.success(app.ID, iface.ID, int64(cost))
		log.Infof("%s used:%dms end success, response:%s", id, cost, rb)
	}

	w.WriteHeader(code)
	w.Write(rb)
}
