package util

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"dearcode.net/crab/log"
	"dearcode.net/crab/validation"
	"github.com/juju/errors"

	"dearcode.net/doodle/pkg/meta"
)

var (
	//AesKey 内部用的简单aes key.
	AesKey = "9F4CrTJJVynV6NJL"
)

var (
	// GitTime git log中记录的提交时间.
	GitTime = ""
	// GitMessage git log 中记录的提交信息.
	GitMessage = ""
)

// PrintVersion 输出当前程序编译信息.
func PrintVersion() {
	fmt.Printf("API GATE\n")
	fmt.Printf("Commit Time: %s\n", GitTime)
	fmt.Printf("Commit Message: %s\n", GitMessage)
}

const (
	backendTimeout = time.Minute
)

// DoRequest 直接发送请求
func DoRequest(req *http.Request) ([]byte, int, error) {
	client := http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, backendTimeout)
				if err != nil {
					log.Errorf("DialTimeout %s:%s", netw, addr)
					return nil, errors.Trace(err)
				}
				deadline := time.Now().Add(backendTimeout)
				if err = c.SetDeadline(deadline); err != nil {
					log.Errorf("SetDeadline %s:%s", netw, addr)
					return nil, errors.Trace(err)
				}
				return c, nil
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, errors.Trace(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, errors.Trace(err)
	}
	return data, resp.StatusCode, nil
}

// Request 调用远程http服务.
func Request(method, url string, headers map[string]string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return DoRequest(req)
}

// DecodeRequestValue 解析request中数据.
func DecodeRequestValue(req *http.Request, result interface{}) error {
	if err := req.ParseForm(); err != nil {
		return err
	}

	rt := reflect.TypeOf(result)
	rv := reflect.ValueOf(result)

	if rt.Kind() == reflect.Ptr && rt.Elem().Kind() == reflect.Struct {
		rt = rt.Elem()
		rv = rv.Elem()
	}

	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		key := f.Tag.Get("json")
		if key == "" {
			key = f.Name
		}
		val := req.FormValue(key)
		switch f.Type.Kind() {
		case reflect.Int, reflect.Int64:
			vi, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				//不需要验证的key就不返回错误了
				if f.Tag.Get("valid") == "" {
					break
				}
				return fmt.Errorf("key:%v value:%v format error", key, val)
			}
			rv.Field(i).SetInt(vi)
		case reflect.String:
			rv.Field(i).SetString(val)
		}
	}

	valid := validation.Validation{}
	_, err := valid.Valid(result)
	return err
}

// SendResponseJSON 返回结果，支持json
func SendResponseJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Add("Content-Type", "application/json")
	r := meta.Response{Data: data}
	buf, _ := json.Marshal(&r)
	log.Debugf("response:%v", string(buf))
	w.Write(buf)
}

// SendResponse 返回结果，支持json
func SendResponse(w http.ResponseWriter, status int, f string, args ...interface{}) {
	w.Header().Add("Content-Type", "application/json")
	r := meta.Response{Status: status, Message: f}
	if len(args) > 0 {
		r.Message = fmt.Sprintf(f, args...)
	}

	buf, _ := json.Marshal(&r)
	log.Debugf("response:%v", string(buf))
	w.Write(buf)
}

// LocalAddr 本机地址
func LocalAddr() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	panic("address not found")
}
