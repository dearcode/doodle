package rbac

import (
	"encoding/binary"
	"fmt"
	"net/http"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/util/aes"

	"dearcode.net/doodle/rbac/config"
	"dearcode.net/doodle/rbac/meta"
	"dearcode.net/doodle/util"
)

type rbacAPP struct {
}

// GET app查询接口.
func (a *rbacAPP) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		AppID  int64  `json:"app_id"`
		Email  string `json:"email"`
		Sort   string `json:"sort"`
		Order  string `json:"order"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	where := fmt.Sprintf("id=%d", vars.AppID)
	if vars.Email != "" {
		where += fmt.Sprintf(" and email='%s'", vars.Email)
	}

	var as []meta.App

	total, err := query("app", where, vars.Sort, vars.Order, vars.Offset, vars.Limit, &as)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Debugf("query error:%s", err.Error())
		return
	}

	if total == 0 || len(as) == 0 {
		server.SendResponse(w, http.StatusNotFound, "not found")
		log.Debugf("App not found, vars:%#v", vars)
		return
	}

	log.Debugf("query App id:%d, Apps:%#v", vars.AppID, as)
	server.SendResponseData(w, as)
}

// POST 添加应用
func (a *rbacAPP) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Comments string `json:"comments"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Debugf("vars:%+v", vars)

	id, token, err := AppAdd(vars.Name, vars.Email, vars.Comments)
	if err != nil {
		log.Errorf("AppAdd vars:%v, err:%v", vars, err)
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("add app:%v, id:%d, token:%s", vars, id, token)
	server.SendResponseData(w, token)
}

// AppAdd 添加app.
func AppAdd(name, email, comments string) (int64, string, error) {
	app := meta.App{
		Name:     name,
		Email:    email,
		Comments: comments,
	}

	id, err := add("app", app)
	if err != nil {
		return 0, "", err
	}

	buf := make([]byte, 8)
	binary.PutVarint(buf, id)

	token, err := aes.Encrypt(string(buf), config.RBAC.Server.Key)
	if err != nil {
		return 0, "", err
	}

	return id, token, updateAppToken(id, token)
}
