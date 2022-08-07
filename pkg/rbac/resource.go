package rbac

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"github.com/juju/errors"

	"dearcode.net/doodle/pkg/rbac/meta"
	"dearcode.net/doodle/pkg/util"
)

type rbacResource struct {
}

// GET 查询指定应用的所有资源
func (res *rbacResource) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID     int    `json:"id"`
		Query  int    `json:"query"`
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

	if vars.Query == 0 {
		var rs []meta.Resource
		if rs, err = ResourceGet(appID, int64(vars.ID)); err != nil {
			server.SendResponse(w, http.StatusBadRequest, err.Error())
			log.Errorf("ResourceGet error:%v, vars:%v", errors.ErrorStack(err), vars)
			return
		}
		server.SendResponseData(w, rs)
		return
	}

	total, rs, err := ResourceQuery(appID, 0, vars.Sort, vars.Order, vars.Offset, vars.Limit)
	if err != nil {
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		log.Debugf("query error:%s", err.Error())
		return
	}

	if total == 0 || len(rs) == 0 {
		server.SendResponse(w, http.StatusNotFound, "not found")
		log.Debugf("Resource not found, vars:%v", vars)
		return
	}

	log.Debugf("appID:%d, query resource:%v", appID, rs)
	server.SendResponseData(w, rs)
}

// POST 为应用添加资源.
func (res *rbacResource) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Name     string `json:"Name"`
		Comments string `json:"Comments"`
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

	id, err := ResourceAdd(appID, vars.Name, vars.Comments)
	if err != nil {
		log.Errorf("add resource:%v, err:%v", vars, err.Error())
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("appID:%d, add resource:%v, id:%d", appID, vars, id)
	server.SendResponseData(w, id)
}

// ResourceAdd 添加资源
func ResourceAdd(appID int64, name, comments string) (int64, error) {
	res := meta.Resource{
		AppID:    appID,
		Name:     name,
		Comments: comments,
	}

	return add("resource", res)
}

// ResourceQuery 查询指定应用的所有资源
func ResourceQuery(appID, resID int64, sort, order string, offset, limit int) (int, []meta.Resource, error) {
	var where string

	if appID > 0 {
		where = fmt.Sprintf("app_id=%d", appID)
	}

	if resID > 0 {
		if where != "" {
			where += " and "
		}
		where += fmt.Sprintf(" id=%d", resID)
	}

	var rs []meta.Resource

	total, err := query("resource", where, sort, order, offset, limit, &rs)
	if err != nil {
		return 0, nil, err
	}

	return total, rs, nil
}

// ResourceGet 查询指定资源信息
func ResourceGet(appID int64, resID ...int64) ([]meta.Resource, error) {
	where := bytes.NewBufferString(fmt.Sprintf("app_id=%d and id in (", appID))
	for _, id := range resID {
		fmt.Fprintf(where, "%d,", id)
	}

	where.Truncate(where.Len() - 1)
	where.WriteString(")")

	var rs []meta.Resource

	if _, err := query("resource", where.String(), "", "", 0, 0, &rs); err != nil {
		return nil, errors.Trace(err)
	}

	return rs, nil
}

// ResourceDelete 删除资源.
func ResourceDelete(resID int64) error {
	//清理资源
	sql := fmt.Sprintf("delete from resource where id=%d", resID)
	a, err := exec(sql)
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("sql:%v, RowsAffected:%v", sql, a)

	//清理关联
	sql = fmt.Sprintf("delete from role_resource where resource_id=%d", resID)
	if a, err = exec(sql); err != nil {
		return errors.Trace(err)
	}
	log.Debugf("sql:%v, RowsAffected:%v", sql, a)
	return nil
}

// DELETE 删除资源.
func (res *rbacResource) DELETE(w http.ResponseWriter, r *http.Request) {
	appID, err := parseToken(r)
	if err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		log.Errorf("parseToken error:%v", errors.ErrorStack(err))
		return
	}

	idStr := r.URL.Query().Get("id")

	log.Debugf("delete app:%v resource:%v", appID, idStr)

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Errorf("strconv parse:%v, err:%v", idStr, err.Error())
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = ResourceDelete(id); err != nil {
		log.Errorf("delete resource:%v, err:%v", id, err.Error())
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Infof("delete resource:%v app:%d", id, appID)

	server.SendResponseOK(w)
}
