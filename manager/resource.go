package manager

import (
	"net/http"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"github.com/juju/errors"

	"dearcode.net/doodle/util"
)

type resource struct {
	ResourceID int64 `json:"resourceID"`
	RoleID     int64 `json:"roleID"`
}

// GET 根据条件查询管理组.
func (res *resource) GET(w http.ResponseWriter, r *http.Request) {
	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, res); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if !u.IsAdmin && res.ResourceID == 0 {
		log.Errorf("%v resource id is 0, vars:%v", r.RemoteAddr, res)
		util.SendResponse(w, http.StatusBadRequest, "resourceID is 0")
		return
	}

	rs, err := rbacClient.GetResourceRoles(res.ResourceID)
	if err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		log.Errorf("query vars:%v error:%s", res, errors.ErrorStack(err))
		return
	}
	log.Debugf("query:%+v, resource:%v", res, rs)
	server.SendData(w, rs)
}

type resourceInfo struct {
	ResourceID int64  `json:"resource_id" validate:"Required"`
	Sort       string `json:"sort"`
	Order      string `json:"order"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
}

// GET 获取资源信息.
func (ri *resourceInfo) GET(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, ri); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	rs, err := rbacClient.GetResource(ri.ResourceID)
	if err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		log.Errorf("query vars:%v error:%s", ri, errors.ErrorStack(err))
		return
	}

	log.Debugf("query:%v, resource:%v", ri, rs)
	server.SendResponseData(w, rs)
}

// POST 关联角色
func (res *resource) POST(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, res); err != nil {
		util.SendResponse(w, 500, err.Error())
		return
	}

	id, err := rbacClient.PostRoleResource(res.RoleID, res.ResourceID)
	if err != nil {
		log.Errorf("RelationResourceRoleAdd error, vars:%v, err:%v", res, err)
		util.SendResponse(w, 500, err.Error())
		return
	}

	log.Debugf("add relation vars:%v, id:%d", res, id)
	util.SendResponseJSON(w, id)
}

type resourceRole struct {
	ID         int64 `json:"id"`
	ResourceID int64 `json:"resourceID"`
	RoleID     int64 `json:"roleID"`
}

// DELETE 解除关联
func (rr *resourceRole) DELETE(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, rr); err != nil {
		util.SendResponse(w, 500, err.Error())
		return
	}

	if err := rbacClient.DeleteResourceRole(rr.ResourceID, rr.RoleID); err != nil {
		log.Errorf("DeleteResourceRole error, vars:%v, err:%v", rr, err)
		util.SendResponse(w, 500, err.Error())
		return
	}

	log.Debugf("del relation vars:%+v", rr)

	util.SendResponseJSON(w, nil)
}
