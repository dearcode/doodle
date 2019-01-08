package main

import (
	"fmt"

	"github.com/dearcode/doodle/service"
)

type echo struct {
}

type echoRequest struct {
	service.RequestHeader
	ID   int    `json:"id" comment:"用户的ID" required:"true"`
	User string `json:"user" comment:"用户名"`
}

type echoResponse struct {
	service.ResponseHeader
	Token string `json:"token" comment:"返回测试的Token"`
}

//Post 根据用户名及ID生成Token.
func (e echo) Post(req echoRequest, resp *echoResponse) {
	resp.Token = fmt.Sprintf("%v_%v", req.ID, req.User)
}

func main() {
	srv := service.New()
	srv.Init()
	srv.Register(echo{})
	srv.Start()
}
