package service

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

type User struct {
}

type UserInfo struct {
	Name  string `json:"name" comment:"用户名"`
	Email string `json:"email" comment:"邮箱地址"`
}

type UserRequest struct {
	RequestHeader
	ID   int
	User UserInfo
}

type UserResponse struct {
	Code    int
	Message string
}

func (u User) Post(req UserRequest, resp *UserResponse) {
	resp.Message = fmt.Sprintf("Post Error, Session:%v", req.Session)
}

func (u User) Get(req UserRequest, resp *UserResponse) {
	fmt.Printf("req:%#v, resp:%v\n", req, resp)
	resp.Code = 998
}

type debugResponseWriter struct {
	http.ResponseWriter
}

func (drw *debugResponseWriter) Header() http.Header {
	return http.Header(make(map[string][]string))
}

func (drw *debugResponseWriter) Write(buf []byte) (int, error) {
	fmt.Printf("response:%s\n", buf)
	return 0, nil
}

func TestRegister(t *testing.T) {
	svc := New()
	svc.Init()
	svc.Register(User{})
	req, _ := http.NewRequest("GET", "http://127.0.0.1:9000/goapi/service/User", bytes.NewBufferString(`{"ID":987}`))
	req.Header.Set("Session", "1111111111111")
	svc.handler(&debugResponseWriter{}, req)
	req, _ = http.NewRequest("POST", "http://127.0.0.1:9000/goapi/service/User", bytes.NewBufferString(`{"ID":654}`))
	req.Header.Set("Session", "222222222222")
	svc.handler(&debugResponseWriter{}, req)
}
