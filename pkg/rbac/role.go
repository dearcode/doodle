package rbac

import (
	"encoding/binary"
	"fmt"
	"net/http"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/util/aes"
	"github.com/juju/errors"

	"dearcode.net/doodle/pkg/rbac/config"
	"dearcode.net/doodle/pkg/rbac/meta"
	"dearcode.net/doodle/pkg/util"
)

type rbacRole struct {
}

// GET 查询指定app的所有role.
func (role *rbacRole) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Query  int    `json:"query"`
		ID     string `json:"role_id"`
		Sort   string `json:"sort"`
		Order  string `json:"order"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	appID, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v, vars:%v", errors.ErrorStack(err), vars)
		return
	}

	where := fmt.Sprintf("app_id=%d", appID)

	var rs []meta.Role

	if vars.Query == 0 {
		where += fmt.Sprintf(" and id=%v", vars.ID)
		if err = queryAll("role", where, &rs); err != nil {
			server.SendResponse(w, http.StatusBadRequest, err.Error())
			log.Errorf("queryAll error:%v, vars:%v", errors.ErrorStack(err), vars)
			return
		}

		log.Debugf("query role app:%d, roles:%#v", appID, rs)
		server.SendResponseData(w, rs)
		return
	}

	total, err := query("role", where, vars.Sort, vars.Order, vars.Offset, vars.Limit, &rs)
	if err != nil {
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		log.Debugf("query error:%s", errors.ErrorStack(err))
		return
	}

	log.Debugf("query role app:%d, roles:%#v", appID, rs)
	server.SendRows(w, total, rs)
}

// POST 为app添加role.
func (role *rbacRole) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Name     string `json:"name"`
		User     string `json:"user"`
		Email    string `json:"email"`
		Comments string `json:"comments"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	appID, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v, vars:%v", errors.ErrorStack(err), vars)
		return
	}

	id, err := RoleAdd(appID, vars.Name, vars.User, vars.Email, vars.Comments)
	if err != nil {
		log.Errorf("RoleAdd vars:%v, error:%s", vars, errors.ErrorStack(err))
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.SendResponseData(w, id)
}

// PUT 修改role信息.
func (role *rbacRole) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID       int64  `json:"role_id"`
		Name     string `json:"name"`
		Comments string `json:"comments"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	appID, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v, vars:%v", errors.ErrorStack(err), vars)
		return
	}

	if err = RoleUpdate(appID, vars.ID, vars.Name, vars.Comments); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("RoleUpdate error:%v, vars:%v", errors.ErrorStack(err), vars)
		return
	}

	server.SendResponseOK(w)
}

// RoleAdd 创建角色.
func RoleAdd(appID int64, name, user, email, comments string) (int64, error) {
	uid, err := UserAdd(appID, user, email)
	if err != nil {
		log.Errorf("UserAdd error:%v, user:%v, email:%v", err.Error(), user, email)
		return 0, err
	}

	role := meta.Role{
		AppID:    appID,
		Name:     name,
		UserID:   uid,
		Comments: comments,
	}

	rid, err := add("role", role)
	if err != nil {
		log.Errorf("add role error:%v, role:%v", err.Error(), role)
		return 0, errors.Trace(err)
	}

	if _, err = RoleUserAdd(appID, rid, user, email); err != nil {
		log.Errorf("RoleUserAdd error:%v, role:%v", err.Error(), role)
		return 0, errors.Trace(err)
	}

	return rid, nil
}

// RoleUpdate 修改角色信息, 需要指定roleID.
func RoleUpdate(appID, roleID int64, name, comments string) error {
	role := meta.Role{
		ID:       roleID,
		AppID:    appID,
		Name:     name,
		Comments: comments,
	}

	return updateRole(role)
}

// RoleGetWithToken 根据token查询role.
func RoleGetWithToken(token, email string) ([]meta.Role, error) {
	buf, err := aes.Decrypt(token, config.RBAC.Server.Key)
	if err != nil {
		return nil, err
	}

	id, n := binary.Varint([]byte(buf))
	if n < 1 {
		return nil, fmt.Errorf("invalid token %s", token)
	}

	where := fmt.Sprintf("app_id=%d and role.user_id=user.id", id)
	if email != "" {
		where += fmt.Sprintf(" and user.email='%s'", email)
	}

	var rs []meta.Role

	if err := queryAll("role", where, &rs); err != nil {
		return nil, errors.Trace(err)
	}

	return rs, nil
}

// RoleQuery 根据条件查找用户有关的role
func RoleQuery(appID, roleID int64, email, sort, order string, offset, limit int) (int, []meta.RoleUser, error) {
	where := "role_user.role_id = role.id and role_user.user_id = user.id"

	if appID > 0 {
		where += fmt.Sprintf(" and role_user.app_id=%d", appID)
	}

	if len(email) > 0 {
		where += fmt.Sprintf(" and user.email='%s'", email)
	}

	if roleID > 0 {
		where += fmt.Sprintf(" and role_user.role_id=%d", roleID)
	}

	var rs []meta.RoleUser

	total, err := query("role_user, role, user", where, sort, order, offset, limit, &rs)
	if err != nil {
		return 0, nil, errors.Trace(err)
	}

	if total == 0 || len(rs) == 0 {
		return 0, nil, nil
	}

	return total, rs, nil
}

// RoleDelete 删除指定role，判断是否存在关联，有关联不能删除
func RoleDelete(appID, roleID int64) error {
	sql := fmt.Sprintf("select id from role_user where app_id=%d and role_id=%d limit 1", appID, roleID)
	ok, err := validate(sql)
	if err != nil {
		return errors.Trace(err)
	}

	if ok {
		return fmt.Errorf("当前角色已关联用户，不能删除")
	}

	sql = fmt.Sprintf("select id from role_resource where app_id=%d and role_id=%d limit 1", appID, roleID)
	ok, err = validate(sql)
	if err != nil {
		return errors.Trace(err)
	}
	if ok {
		return fmt.Errorf("当前角色存在关联资源，不能删除")
	}

	sql = fmt.Sprintf("delete from role where id=%d and app_id=%d", roleID, appID)
	a, err := exec(sql)
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("sql:%v, RowsAffected:%v", sql, a)

	return nil
}

// DELETE 删除role.
func (role *rbacRole) DELETE(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID   int64  `json:"role_id"`
		Name string `json:"name"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	appID, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v", errors.ErrorStack(err))
		return
	}

	var where string

	if vars.Name != "" {
		where = fmt.Sprintf("app_id=%d and role_id=(select id from role where app_id=%d and name='%s')", appID, appID, vars.Name)
	}

	if vars.ID != 0 {
		where = fmt.Sprintf("app_id=%d and role_id=%d", appID, vars.ID)
	}

	sql := "delete from role_user where " + where
	a, err := exec(sql)
	if err != nil {
		log.Errorf("exec sql:%v, error:%v", sql, errors.ErrorStack(err))
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Debugf("sql:%v, RowsAffected:%v", sql, a)

	if vars.Name != "" {
		sql = fmt.Sprintf("delete from role where app_id=%d and name='%s'", appID, vars.Name)
	}

	if vars.ID != 0 {
		sql = fmt.Sprintf("delete from role where id=%d", vars.ID)
	}

	if a, err = exec(sql); err != nil {
		log.Errorf("exec sql:%v, error:%v", sql, errors.ErrorStack(err))
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Debugf("sql:%v, RowsAffected:%v", sql, a)
	server.SendResponseOK(w)
}
