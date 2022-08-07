package rbac

import (
	"fmt"
	"net/http"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/orm"
	"github.com/juju/errors"

	"dearcode.net/doodle/rbac/meta"
	"dearcode.net/doodle/util"
)

type rbacUser struct {
}

func (ru *rbacUser) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Sort   string `json:"sort"`
		Order  string `json:"order"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var where string

	appID, _ := parseToken(r)
	if appID != 0 {
		where = fmt.Sprintf("id=%d", appID)
	}

	var us []meta.User

	total, err := query("user", where, vars.Sort, vars.Order, vars.Offset, vars.Limit, &us)
	if err != nil {
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		log.Debugf("query error:%s", err.Error())
		return
	}

	if total == 0 || len(us) == 0 {
		server.SendResponse(w, http.StatusNotFound, "not found")
		log.Debugf("User not found, vars:%#v", vars)
		return
	}

	log.Debugf("query appID:%d, User:%#v", appID, us)
	server.SendResponseData(w, us)
}

func (ru *rbacUser) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID    int64  `json:"user_id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Debugf("vars:%+v", vars)

	appID, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v, vars:%v", errors.ErrorStack(err), vars)
		return
	}

	if err = UserUpdate(appID, vars.ID, vars.Name, vars.Email); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("UserUpdate error:%v, vars:%v", errors.ErrorStack(err), vars)
		return
	}

	log.Debugf("UserUpdate:%+v", vars)

	server.SendResponseOK(w)
}

// UserUpdate 修改用户信息
func UserUpdate(appID, userID int64, name, email string) error {
	u := meta.User{
		ID:    userID,
		AppID: appID,
		Name:  name,
		Email: email,
	}

	return errors.Trace(updateUser(u))
}

// POST 用户添加.
func (ru *rbacUser) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Debugf("vars:%v", vars)

	appID, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v, vars:%v", errors.ErrorStack(err), vars)
		return
	}

	id, err := UserAdd(appID, vars.Name, vars.Email)
	if err != nil {
		log.Errorf("UserAdd err:%v", err.Error())
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("UserAdd %v, id:%v", vars, id)

	server.SendResponseData(w, id)
}

// UserAdd 添加用户
func UserAdd(appID int64, name, email string) (int64, error) {
	u := meta.User{
		AppID: appID,
		Name:  name,
		Email: email,
	}

	id, err := add("user", u)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// UserGetAll 根据appID查询用户.
func UserGetAll(appID int64) ([]meta.User, error) {
	where := fmt.Sprintf("id=%d", appID)
	var us []meta.User

	if err := queryAll("user", where, &us); err != nil {
		return nil, err
	}

	return us, nil
}

type rbacUserInfo struct {
}

func (ru *rbacUserInfo) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Email string `json:"email"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("get db connection error:%v", errors.ErrorStack(err))
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	var u meta.User

	if err = orm.NewStmt(db, "user").Where("email='%s'", vars.Email).Query(&u); err != nil {
		log.Errorf("db query user:%s error:%v", vars.Email, errors.ErrorStack(err))
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("query User:%v", u)
	server.SendResponseData(w, u)
}
