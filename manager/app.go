package manager

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"dearcode.net/crab/log"
	"dearcode.net/crab/util/aes"
	"github.com/juju/errors"

	"dearcode.net/doodle/meta"
	"dearcode.net/doodle/util"
)

type application struct {
	ID    int64
	Name  string
	User  string
	Email string
}

type appInfos struct {
	ID    int64  `json:"interfaceID"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Sort  string `json:"sort"`
	Order string `json:"order"`
	Page  int    `json:"offset"`
	Size  int    `json:"limit"`
}

// GET 获取未授权应用基本信息
func (ais *appInfos) GET(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, ais); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	apps := []application{}

	var where string

	if ais.Email != "" {
		if where != "" {
			where += " and "
		}
		where += fmt.Sprintf("application.email like '%%%s%%'", ais.Email)
	}

	if ais.Name != "" {
		if where != "" {
			where += " and "
		}
		where += fmt.Sprintf("application.name like '%%%s%%'", ais.Name)
	}

	if where != "" {
		where += " and "
	}

	where += fmt.Sprintf("application.id not in (select application_id from relation where relation.interface_id = %d)", ais.ID)

	total, err := query("application", where, ais.Sort, ais.Order, ais.Page, ais.Size, &apps)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	if len(apps) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total":0,"rows":[]}`))
		log.Debugf("service not found")
		return
	}

	buf, err := json.Marshal(apps)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"total":%d, "rows":%s}`, total, buf)))
}

type appInfo struct {
	ID int64 `json:"id"`
}

func (ai *appInfo) GET(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, ai); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	p, err := getApp(ai.ID)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	buf, err := json.Marshal(p)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

type app struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	User    string `json:"user"`
	Comment string `json:"comment"`
	CTime   string `db_default:"now()"`
	Mtime   string `db_default:"now()"`
}

func (a *app) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Sort  string `json:"sort"`
		Order string `json:"order"`
		Page  int    `json:"offset"`
		Size  int    `json:"limit"`
	}{}
	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if err = util.DecodeRequestValue(r, &vars); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	var apps []meta.Application
	var where string

	if !u.IsAdmin {
		where = fmt.Sprintf("application.email='%s'", u.Email)
	}

	if vars.Name != "" {
		if where != "" {
			where += " and "
		}
		where += fmt.Sprintf(" application.name like '%%%s%%'", vars.Name)
	}

	if vars.Email != "" {
		if where != "" {
			where += " and "
		}
		where += fmt.Sprintf(" application.email like '%%%s%%'", vars.Email)
	}

	total, err := query("application", where, vars.Sort, vars.Order, vars.Page, vars.Size, &apps)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	if len(apps) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total":0,"rows":[]}`))
		log.Debugf("service not found")
		return
	}

	//TODO test
	for _, sa := range apps {
		sa.Token = ""
	}

	result := struct {
		Total int                `json:"total"`
		Rows  []meta.Application `json:"rows"`
	}{total, apps}

	buf, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (a *app) DELETE(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id"`
	}{}
	if err := util.DecodeRequestValue(r, vars); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := del("application", vars.ID); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponse(w, 0, "")

	log.Debugf("delete service:%v, success", vars.ID)
}

func (a *app) POST(w http.ResponseWriter, r *http.Request) {
	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if err = util.DecodeRequestValue(r, a); err != nil {
		log.Errorf("DecodeRequestValue req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !u.IsAdmin {
		a.User = u.User
		a.Email = u.Email
	}

	id, err := add("application", a)
	if err != nil {
		if strings.Contains(err.Error(), "1062") {
			log.Errorf("add req:%+v, error:%s", r, errors.ErrorStack(err))
			util.SendResponse(w, http.StatusInternalServerError, "已存在同名应用")
			return
		}

		log.Errorf("add req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	buf := make([]byte, 8)
	binary.PutVarint(buf, id)

	token, err := aes.Encrypt(string(buf), util.AesKey)
	if err != nil {
		log.Errorf("AesEncrypt req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = updateAppToken(id, token); err != nil {
		log.Errorf("updateAppToken req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	util.SendResponse(w, 0, fmt.Sprintf(`{"id":%d}`, id))

	log.Debugf("add service success, id:%v, token:%s", id, token)
}

func (a *app) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID      int64  `json:"id" valid:"Required"`
		Name    string `json:"name"  valid:"Required"`
		User    string `json:"user"  valid:"Required"`
		Email   string `json:"email" valid:"Email"`
		Comment string `json:"comment"  valid:"Required"`
	}{}
	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := util.DecodeRequestValue(r, vars); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	//非管理员帐号，不允许更改邮箱
	if !u.IsAdmin {
		if vars.Email != u.Email {
			log.Errorf("isAdmin:%v, email:%s, erp email:%s", u.IsAdmin, vars.Email, u.Email)
			util.SendResponse(w, http.StatusInternalServerError, "不能修改别人添加的应用")
			return
		}
	}

	if err := updateApp(fmt.Sprintf("id=%d", vars.ID), vars.Name, vars.User, vars.Email, vars.Comment); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponse(w, 0, "")

	log.Debugf("update service success, new:%+v", vars)
}
