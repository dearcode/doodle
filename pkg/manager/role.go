package manager

import (
	"net/http"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"github.com/juju/errors"

	"dearcode.net/doodle/pkg/util"
)

type roleInfo struct {
}

func (ri *roleInfo) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		RoleID int64 `json:"role_id"`
	}{}
	_, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	ro, err := rbacClient.GetRole(vars.RoleID)
	if err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		log.Errorf("RoleInfoGet vars:%v error:%s", vars, errors.ErrorStack(err))
		return
	}

	log.Debugf("query:%v, role:%v", vars, ro)
	util.SendResponseJSON(w, ro)
}

type role struct {
	RoleID  int64  `json:"role_id"`
	Name    string `json:"name"`
	User    string `json:"user"`
	Email   string `json:"email"`
	Comment string `json:"comment"`
}

func (r *role) GET(w http.ResponseWriter, req *http.Request) {
	vars := struct {
		ResourceID int64  `json:"resource_id"`
		Email      string `json:"email"`
	}{}

	u, err := session.User(w, req)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), req)
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(req, &vars); err != nil {
		log.Errorf("DecodeRequestValue error:%v", err)
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	//不是管理员的话只能使用erp中信息
	if !u.IsAdmin {
		vars.Email = u.Email
	}

	rs, err := rbacClient.GetResourceRolesUnrelated(vars.ResourceID, vars.Email)
	if err != nil {
		log.Errorf("GetResourceRolesUnrelated error, vars:%v, err:%v", r, err)
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("unrelated role %#v, id:%d", rs, vars.ResourceID)

	server.SendData(w, rs)
}

func (r *role) POST(w http.ResponseWriter, req *http.Request) {
	u, err := session.User(w, req)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), req)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	if err = util.DecodeRequestValue(req, r); err != nil {
		util.SendResponse(w, 500, err.Error())
		return
	}

	//不是管理员的话只能使用erp中信息
	if !u.IsAdmin {
		r.User = u.User
		r.Email = u.Email
	}

	id, err := rbacClient.PostRole(r.Name, r.Comment, r.User, r.Email)
	if err != nil {
		log.Errorf("RoleAdd error, vars:%v, err:%v", r, err)
		util.SendResponse(w, 500, err.Error())
		return
	}

	log.Debugf("add role %v, id:%d", r, id)
	util.SendResponseJSON(w, id)

}

func (r *role) PUT(w http.ResponseWriter, req *http.Request) {
	u, err := session.User(w, req)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), req)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(req, r); err != nil {
		util.SendResponse(w, 500, err.Error())
		return
	}

	//不是管理员的话只能使用erp中信息
	if !u.IsAdmin {
		r.User = u.User
		r.Email = u.Email
	}

	if err := rbacClient.PutRole(r.RoleID, r.Name, r.Comment); err != nil {
		log.Errorf("RoleUpdate error, vars:%v, err:%v", r, err)
		util.SendResponse(w, 500, err.Error())
		return
	}

	log.Debugf("update role %v", r)
	util.SendResponseJSON(w, nil)
}

type roleUser struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	RoleID int64  `json:"role_id"`
	Sort   string `json:"sort"`
	Order  string `json:"order"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

func (ru *roleUser) PUT(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, ru); err != nil {
		util.SendResponse(w, 500, err.Error())
		return
	}

	if err := rbacClient.PutUser(ru.UserID, ru.Name, ru.Email); err != nil {
		log.Errorf("UserUpdate vars:%v, err:%v", ru, err)
		util.SendResponse(w, 500, err.Error())
		return
	}

	log.Debugf("update user:%+v", ru)

	util.SendResponseJSON(w, nil)

}

func (ru *roleUser) POST(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, ru); err != nil {
		util.SendResponse(w, 500, err.Error())
		return
	}

	id, err := rbacClient.PostRoleUser(ru.RoleID, ru.Name, ru.Email)
	if err != nil {
		log.Errorf("PostRoleUser error, vars:%v, err:%v", ru, err)
		util.SendResponse(w, 500, err.Error())
		return
	}

	log.Debugf("add role %+v, id:%d", ru, id)

	util.SendResponseJSON(w, id)
}

func (ru *roleUser) GET(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, ru); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	switch ru.Sort {
	case "RoleName":
		ru.Sort = "role.name"
	case "RoleComment":
		ru.Sort = "role.comment"
	case "Mtime":
		ru.Sort = "role_user.mtime"
	}

	rs, err := rbacClient.GetRoleUsers(ru.RoleID)
	if err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		log.Errorf("query vars:%v error:%s", ru, errors.ErrorStack(err))
		return
	}

	if len(rs) == 0 {
		util.SendResponse(w, http.StatusNotFound, "not found")
		log.Debugf("role not found, vars:%v", ru)
		return
	}

	log.Debugf("query:%v, role:%v", ru, rs)
	util.SendResponseJSON(w, rs)
}

type userRole struct {
	Sort   string `json:"sort"`
	Order  string `json:"order"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

func (ur *userRole) GET(w http.ResponseWriter, r *http.Request) {
	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, ur); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	switch ur.Sort {
	case "RoleName":
		ur.Sort = "role.name"
	case "RoleComment":
		ur.Sort = "role.comment"
	case "Mtime":
		ur.Sort = "role_user.mtime"
	}

	rs, err := rbacClient.GetUserRoles(u.Email)
	if err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		log.Errorf("query vars:%v error:%s", ur, errors.ErrorStack(err))
		return
	}

	if len(rs) == 0 {
		util.SendResponse(w, http.StatusNotFound, "not found")
		log.Debugf("role not found, vars:%v", ur)
		return
	}

	log.Debugf("query:%v, role:%v", ur, rs)
	server.SendData(w, rs)
}

// DELETE 删除角色
func (r *role) DELETE(w http.ResponseWriter, req *http.Request) {
	if err := util.DecodeRequestValue(req, r); err != nil {
		util.SendResponse(w, 500, err.Error())
		return
	}

	if err := rbacClient.DeleteRole(r.RoleID, ""); err != nil {
		log.Errorf("RoleDelete error, vars:%v, err:%v", r, errors.ErrorStack(err))
		util.SendResponse(w, 500, err.Error())
		return
	}

	log.Debugf("del role vars:%v", r)

	util.SendResponseJSON(w, nil)
}

// onRoleUserDelete 删除角色中的用户
func (ru *roleUser) DELETE(w http.ResponseWriter, r *http.Request) {
	_, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, ru); err != nil {
		util.SendResponse(w, 500, err.Error())
		return
	}

	if err := rbacClient.DeleteRoleUser(ru.RoleID, ru.Email); err != nil {
		log.Errorf("RoleDelete error, vars:%v, err:%v", ru, errors.ErrorStack(err))
		util.SendResponse(w, 500, err.Error())
		return
	}

	log.Debugf("del role user vars:%v", ru)

	util.SendResponseJSON(w, nil)
}
