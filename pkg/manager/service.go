package manager

import (
	"fmt"
	"net/http"
	"strings"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/orm"
	"github.com/juju/errors"

	"dearcode.net/doodle/pkg/meta"
	"dearcode.net/doodle/pkg/util"
)

type serviceInfo struct {
	ID int64 `json:"id"`
}

func (pi *serviceInfo) GET(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, pi); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	var p meta.Service

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	if err = orm.NewStmt(db, "service").Where("id=%d", pi.ID).Query(&p); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	server.SendData(w, p)
}

type service struct {
}

func (p *service) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
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

	var where string
	if !u.IsAdmin {
		where = fmt.Sprintf(" service.resource_id in (%s)", u.ResKey)
	}

	var ps []meta.Service

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	stmt := orm.NewStmt(db, "service").Where(where)
	total, err := stmt.Count()
	if err != nil {
		log.Errorf("Count req:%+v, error:%v", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "查询数据库出错")
		return
	}

	if total == 0 {
		log.Infof("service not found,req:%+v", r)
		server.SendRows(w, 0, nil)
		return
	}

	if err = stmt.Order(vars.Order).Offset(vars.Page).Limit(vars.Size).Sort(vars.Sort).Query(&ps); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	server.SendRows(w, total, ps)
}

func (p *service) DELETE(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := del("service", vars.ID); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponse(w, 0, "")

	log.Debugf("delete service:%v, success", vars.ID)
}

func (p *service) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		meta.Service
	}{}

	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = server.ParseFormVars(r, &vars); err != nil {
		log.Errorf("invalid request:%v, error:%v", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	fmt.Printf("vars:%#v\n", vars)

	if !u.IsAdmin {
		vars.Email = u.Email
		vars.User = u.User
	}

	resID, err := rbacClient.PostResource(vars.Name, vars.Comment)
	if err != nil {
		log.Errorf("ResourceAdd req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "添加资源出错")
		return
	}

	roleID, err := rbacClient.PostRole(vars.Name, "默认添加的管理组", vars.User, vars.Email)
	if err != nil {
		log.Errorf("RoleAdd req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "添加角色出错")
		return
	}

	if _, err = rbacClient.PostRoleResource(roleID, resID); err != nil {
		log.Errorf("RelationResourceRoleAdd req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "为项目授权角色出错")
		return
	}

	vars.ResourceID = resID
	vars.RoleID = roleID

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	id, err := orm.NewStmt(db, "service").Insert(vars.Service)
	if err != nil {
		log.Errorf("add req:%+v, error:%s", r, errors.ErrorStack(err))
		if strings.Contains(err.Error(), "1062") {
			util.SendResponse(w, http.StatusInternalServerError, "项目路径已存在, 项目路径在接口平台中是唯一的，不能重用")
			return
		}
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	var scr []interface{}

	for _, cid := range vars.ClusterIDs {
		scr = append(scr, &serviceClusterRelation{
			ClusterID: cid,
			ServiceID: id,
		})
	}

	if _, err = orm.NewStmt(db, "service_cluster").SetLogger(log.GetLogger()).BatchInsert(scr); err != nil {
		log.Errorf("add service_cluster req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	util.SendResponse(w, 0, fmt.Sprintf(`{"id":%d}`, id))

	log.Debugf("add service:%v, id:%v, role:%d, resource:%d", vars, id, roleID, resID)
}

type serviceClusterRelation struct {
	ServiceID int64 `db:"service_id"`
	ClusterID int64 `db:"cluster_id"`
}

func (p *service) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID        int64  `json:"id" valid:"Required"`
		Name      string `json:"name"  valid:"Required"`
		User      string `json:"user"  valid:"Required"`
		Email     string `json:"email"  valid:"Email"`
		Path      string `json:"path"  valid:"AlphaNumeric"`
		ClusterID int64  `json:"cluster_id"`
		Source    string `json:"source"`
		Version   int    `json:"version"`
		Comment   string `json:"comment"  valid:"Required"`
	}{}
	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !u.IsAdmin {
		vars.Email = u.Email
		vars.User = u.User
	}

	db, err := mdb.GetConnection()
	if err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	ra, err := orm.NewStmt(db, "service").Where("id=%v", vars.ID).Update(&vars)
	if err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	util.SendResponse(w, 0, "")

	log.Debugf("update service success:%v, new:%+v", ra, vars)
}

func getServiceResourceID(serviceID int64) (int64, error) {
	return getResourceID("service", serviceID)
}
