package rbac

import (
	"fmt"
	"net/http"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"github.com/juju/errors"

	"dearcode.net/doodle/rbac/meta"
	"dearcode.net/doodle/util"
)

type rbacRoleResource struct {
}

// DELETE 删除角色与资源关系.
func (rr *rbacRoleResource) DELETE(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ResourceID int64 `json:"resource_id"`
		RoleID     int64 `json:"role_id"`
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

	if err = RoleResourceDelete(appID, vars.RoleID, vars.ResourceID); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("RoleResourceDelete error:%v, app:%v vars:%v", errors.ErrorStack(err), appID, vars)
		return
	}

	log.Infof("app:%v resource:%v role:%v success", appID, vars.ResourceID, vars.RoleID)
	server.SendResponseOK(w)
}

//GET 查询角色与资源关系.
func (rr *rbacRoleResource) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		API        int    `json:"api"`
		ResourceID int64  `json:"resource_id"`
		RoleID     int64  `json:"role_id"`
		Sort       string `json:"sort"`
		Order      string `json:"order"`
		Offset     int    `json:"offset"`
		Limit      int    `json:"limit"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	appID, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v, vars:%+v", errors.ErrorStack(err), vars)
		return
	}

	total, rs, err := RoleResourceQuery(appID, vars.ResourceID, vars.RoleID, vars.Sort, vars.Order, vars.Offset, vars.Limit)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("RelationGet error:%v, vars:%+v", err, vars)
		return
	}

	log.Infof("vars:%+v", vars)

	if vars.API == 1 {
		server.SendResponseData(w, rs)
		return
	}

	server.SendRows(w, total, rs)
}

type rbacRoleUser struct {
}

//GET 查询角色对应用户.
func (ru *rbacRoleUser) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Query  int    `json:"query"`
		Email  string `json:"email"`
		RoleID int64  `json:"role_id"`
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

	total, rs, err := RelationRoleUserQuery(appID, vars.RoleID, vars.Email, vars.Sort, vars.Order, vars.Offset, vars.Limit)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("RelationGet error:%v, vars:%v", err, vars)
		return
	}

	if len(rs) == 0 {
		server.SendResponse(w, http.StatusNotFound, "not found")
		log.Infof("RelationGet not found, vars:%v", vars)
		return
	}

	if vars.Query == 0 {
		server.SendResponseData(w, rs)
		return
	}
	server.SendRows(w, total, rs)
}

//POST 添加关联.
func (ru *rbacRoleUser) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Name   string `json:"name"`
		Email  string `json:"email"`
		RoleID int64  `json:"role_id"`
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
	log.Debugf("vars:%v", vars)

	id, err := RoleUserAdd(appID, vars.RoleID, vars.Name, vars.Email)
	if err != nil {
		log.Errorf("RoleUserAdd err:%v, vars:%v", err, vars)
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("add relation %v, id:%d", vars, id)
	server.SendResponseData(w, id)
}

// RoleUserAdd 为角色添加用户.
func RoleUserAdd(appID, roleID int64, name, email string) (int64, error) {
	uid, err := UserAdd(appID, name, email)
	if err != nil {
		return 0, err
	}

	relation := meta.RoleUser{
		AppID:  appID,
		RoleID: roleID,
		UserID: uid,
	}

	log.Debugf("app:%d, name:%v, email:%v, ID:%d", appID, name, email, uid)
	id, err := add("role_user", relation)
	if err != nil {
		return 0, err
	}
	log.Debugf("role:%v, id:%d", relation, id)

	return id, nil
}

//POST 添加关联.
func (rr *rbacRoleResource) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		RoleID     int64 `json:"role_id"`
		ResourceID int64 `json:"resource_id"`
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

	relation := meta.RoleResource{
		AppID:      appID,
		RoleID:     vars.RoleID,
		ResourceID: vars.ResourceID,
	}

	id, err := add("role_resource", relation)
	if err != nil {
		log.Errorf("add role_resource error:%v", err.Error())
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("add relation:%v, id:%d", relation, id)
	server.SendResponseData(w, id)
}

// RoleResourceQuery 根据AppID, ResourceID, RoleID查找关系, 带分页功能.
func RoleResourceQuery(appID, resID, roleID int64, sort, order string, offset, limit int) (int, []meta.RoleResource, error) {
	where := "role_resource.role_id = role.id and role_resource.resource_id = resource.id"

	if appID > 0 {
		where += fmt.Sprintf(" and role_resource.app_id=%d", appID)
	}

	if resID > 0 {
		where += fmt.Sprintf(" and resource.id=%d", resID)
	}

	if roleID > 0 {
		where += fmt.Sprintf(" and role.id=%d", roleID)
	}

	var rs []meta.RoleResource

	total, err := query("role_resource, role, resource", where, sort, order, offset, limit, &rs)
	if err != nil {
		return 0, nil, err
	}

	if total == 0 || len(rs) == 0 {
		return 0, nil, nil
	}

	return total, rs, nil
}

// RelationUserQuery 查询指定role的所有用户, 带分页功能.
func RelationUserQuery(appID, roleID int64, sort, order string, offset, limit int) (int, []meta.RoleUser, error) {
	where := "role_user.role_id = role.id and role_user.user_id = user.id"

	if appID > 0 {
		where += fmt.Sprintf(" and role_user.app_id=%d", appID)
	}

	if roleID > 0 {
		where += fmt.Sprintf(" and role_user.role_id=%d", roleID)
	}

	var rs []meta.RoleUser

	total, err := query("role_user, role, user", where, sort, order, offset, limit, &rs)
	if err != nil {
		return 0, nil, err
	}

	if total == 0 || len(rs) == 0 {
		return 0, nil, nil
	}

	return total, rs, nil
}

// RelationRoleQuery 查询指定用户所在的role, 带分页功能.
func RelationRoleQuery(appID int64, email, sort, order string, offset, limit int) (int, []meta.RoleUser, error) {
	where := "role_user.role_id = role.id and role_user.user_id = user.id"

	if appID > 0 {
		where += fmt.Sprintf(" and role_user.app_id=%d", appID)
	}

	if email != "" {
		where += fmt.Sprintf(" and user.email='%s'", email)
	}

	var rs []meta.RoleUser

	total, err := query("role_user, role, user", where, sort, order, offset, limit, &rs)
	if err != nil {
		return 0, nil, err
	}

	if total == 0 || len(rs) == 0 {
		return 0, nil, nil
	}

	return total, rs, nil
}

// RelationRoleUserQuery 根据AppID, email, RoleID查找关系, 带分页功能.
func RelationRoleUserQuery(appID, roleID int64, email, sort, order string, offset, limit int) (int, []meta.RoleUser, error) {
	where := "role_user.role_id = role.id and role_user.user_id = user.id"

	if appID > 0 {
		where += fmt.Sprintf(" and role_user.app_id=%d", appID)
	}

	if roleID > 0 {
		where += fmt.Sprintf(" and role_user.role_id=%d", roleID)
	}

	if email != "" {
		where += fmt.Sprintf(" and user.email='%s'", email)
	}

	var rs []meta.RoleUser

	total, err := query("role_user, role, user", where, sort, order, offset, limit, &rs)
	if err != nil {
		return 0, nil, err
	}

	if total == 0 || len(rs) == 0 {
		return 0, nil, nil
	}

	return total, rs, nil
}

// UnrelatedResourceRoles 根据AppID, ResourceID, email查找未关联的roles, 带分页功能.
func UnrelatedResourceRoles(appID, resourceID int64, email, sort, order string, offset, limit int) (int, []meta.Role, error) {
	where := fmt.Sprintf("id NOT in (select role_resource.role_id from role_resource where role_resource.resource_id = %d)", resourceID)
	if email != "" {
		where += fmt.Sprintf(" and role.user_id = (select id from user where user.email = '%s') ", email)
	}

	var rs []meta.Role

	total, err := query("role", where, sort, order, offset, limit, &rs)
	if err != nil {
		return 0, nil, err
	}

	if total == 0 || len(rs) == 0 {
		return 0, nil, nil
	}

	return total, rs, nil
}

//RoleUserGet 按role，email查询关联信息.
func RoleUserGet(appID, roleID int64, email string) ([]meta.RoleUser, error) {
	where := "role_user.role_id = role.id and role_user.user_id = user.id"

	if appID > 0 {
		where += fmt.Sprintf(" and role_user.app_id=%d", appID)
	}

	if len(email) > 0 {
		where += fmt.Sprintf(" and user.email='%s'", email)
	}

	if roleID > 0 {
		where += fmt.Sprintf(" and role_id=%d", roleID)
	}

	var rs []meta.RoleUser

	if err := queryAll("role_user, role, user", where, &rs); err != nil {
		return nil, errors.Trace(err)
	}

	return rs, nil
}

// RelationResourceRoleAdd 把资源授权给角色
func RelationResourceRoleAdd(appID, resourceID, roleID int64) (int64, error) {
	relation := meta.RoleResource{
		AppID:      appID,
		RoleID:     roleID,
		ResourceID: resourceID,
	}

	return add("role_resource", relation)
}

// RelationResourceRoleDel 删除授权
func RelationResourceRoleDel(appID, resID, roleID int64) error {
	sql := fmt.Sprintf("delete from role_resource where app_id=%d and resource_id=%d and role_id=%d", appID, resID, roleID)
	a, err := exec(sql)
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("sql:%v, RowsAffected:%v", sql, a)
	return nil
}

// RelationUserRoleDel 删除角色中的指定用户, 如果用户是这个角色的添加者可以删除角色关联的用户，或者是管理员
func RelationUserRoleDel(appID, roleID int64, email, owner string) error {
	if owner != "" {
		sql := fmt.Sprintf("select id from role where id=%d and user_id=(select id from user where email='%s')", roleID, owner)
		ok, err := validate(sql)
		if err != nil {
			return errors.Trace(err)
		}
		if !ok {
			log.Infof("sql:%v", sql)
			return fmt.Errorf("user:%v is not role owner", email)
		}
	}
	sql := fmt.Sprintf("delete from role_user where app_id=%d and role_id=%d and user_id=(select id from user where email='%s')", appID, roleID, email)
	a, err := exec(sql)
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("sql:%v, RowsAffected:%v", sql, a)
	return nil
}

//RelationValidate 权限验证
func RelationValidate(appID, resID int64, email string) error {
	sql := fmt.Sprintf("select id from role_resource where app_id=%d and resource_id=%d and role_id in (select role_id from role_user where user_id = (select id from user where email='%s'))", appID, resID, email)
	ok, err := validate(sql)
	if err != nil {
		return errors.Trace(err)
	}
	if !ok {
		return fmt.Errorf("relation not exist")
	}
	return nil
}

//UserResourceGet 根据用户邮件地址获取资源.
func UserResourceGet(appID int64, email string) (result []int64, err error) {
	where := fmt.Sprintf("app_id=%d and role_id in  (select role_id from role_user where role_user.user_id = (select id from user where email='%s' and app_id=%d))", appID, email, appID)
	vars := []struct {
		ID int64 `db:"DISTINCT role_resource.resource_id"`
	}{}

	if err = queryAll("role_resource", where, &vars); err != nil {
		return
	}

	for _, v := range vars {
		result = append(result, v.ID)
	}

	return
}

type userResource struct {
}

//GET 查询用户资源列表.
func (ur *userResource) GET(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")

	id, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v", errors.ErrorStack(err))
		return
	}

	result, err := UserResourceGet(id, email)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("UserResourceGet error:%v", errors.ErrorStack(err))
		return
	}

	if len(result) == 0 {
		server.SendResponseData(w, []meta.Resource{})
		return
	}

	res, err := ResourceGet(id, result...)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("ResourceGet error:%v", errors.ErrorStack(err))
		return
	}

	server.SendResponseData(w, res)
}

//DELETE 删除关联.
func (ru *rbacRoleUser) DELETE(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Email  string `json:"email"`
		RoleID int64  `json:"role_id"`
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

	log.Debugf("vars:%v", vars)

	if err = RoleUserDelete(appID, vars.RoleID, vars.Email); err != nil {
		log.Errorf("RoleUserDelete err:%v, vars:%v", err, vars)
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("delete role:%v user:%v", vars.RoleID, vars.Email)
	server.SendResponseOK(w)
}

//RoleUserDelete 按role，email查询关联信息.
func RoleUserDelete(appID, roleID int64, email string) error {
	sql := fmt.Sprintf("delete from role_user where app_id=%d and role_id=%d and user_id = (select id from user where app_id=%d and email='%v')", appID, roleID, appID, email)
	a, err := exec(sql)
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("sql:%v, RowsAffected:%v", sql, a)
	return nil
}

//RoleResourceDelete 按role，resource查询关联信息.
func RoleResourceDelete(appID, roleID, resourceID int64) error {
	sql := fmt.Sprintf("delete from role_resource where app_id=%d and role_id=%d and resource_id = %v", appID, roleID, resourceID)
	a, err := exec(sql)
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("sql:%v, RowsAffected:%v", sql, a)
	return nil
}

type userRole struct {
}

//GET 查询用户与角色对应关系.
func (ur *userRole) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Query  int    `json:"query"`
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

	appID, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v, vars:%v", errors.ErrorStack(err), vars)
		return
	}

	total, rs, err := RelationRoleUserQuery(appID, 0, vars.Email, vars.Sort, vars.Order, vars.Offset, vars.Limit)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("RelationGet error:%v, vars:%v", err, vars)
		return
	}

	if len(rs) == 0 {
		server.SendResponse(w, http.StatusNotFound, "not found")
		log.Infof("RelationGet not found, vars:%v", vars)
		return
	}

	if vars.Query == 0 {
		server.SendResponseData(w, rs)
		return
	}

	server.SendRows(w, total, rs)
}

type resourceRolesUnrelated struct {
}

func (rru *resourceRolesUnrelated) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Query      int    `json:"query"`
		ResourceID int64  `json:"resource_id"`
		Email      string `json:"email"`
		Sort       string `json:"sort"`
		Order      string `json:"order"`
		Offset     int    `json:"offset"`
		Limit      int    `json:"limit"`
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

	total, rs, err := UnrelatedResourceRoles(appID, vars.ResourceID, vars.Email, vars.Sort, vars.Order, vars.Offset, vars.Limit)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("UnrelatedResourceRoles error:%v, vars:%v", err, vars)
		return
	}

	if len(rs) == 0 {
		server.SendResponse(w, http.StatusNotFound, "not found")
		log.Infof("UnrelatedResourceRoles not found, vars:%v", vars)
		return
	}

	if vars.Query == 0 {
		server.SendResponseData(w, rs)
		return
	}

	server.SendRows(w, total, rs)
}
