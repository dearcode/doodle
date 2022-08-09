package manager

import (
	"fmt"
	"net/http"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/orm"
	"github.com/juju/errors"

	"dearcode.net/doodle/util"
)

type node struct {
	ID      int64
	Server  string
	Comment string
	Ctime   string `db_default:"now()"`
	Mtime   string `db_default:"now()"`
}

// cluster 集群基本信息.
type cluster struct {
	ID             int64
	Name           string
	ServerUser     string
	ServerPassword string
	ServerKey      string
	RoleID         int64
	User           string
	Email          string
	Comment        string
	Node           []node `db_table:"one2more"`
	Ctime          string `db_default:"auto"`
	Mtime          string `db_default:"auto"`
}

func (c *cluster) GET(w http.ResponseWriter, r *http.Request) {
	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	var cs []cluster
	if err = orm.NewStmt(db, "cluster").Where("role_id in (%v)", u.RolesKey).Query(&cs); err != nil {
		log.Errorf("Query cluster req:%+v, user:%+v error:%s", r, u, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "query db error")
		return
	}

	log.Debugf("user:%v, cluster:%v", u, cs)
	server.SendData(w, cs)
}

func (c *cluster) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID             int64  `json:"id"`
		Name           string `json:"name"`
		User           string `json:"user"`
		RoleID         int64  `json:"role"`
		Email          string `json:"email"`
		ServerUser     string `json:"server_user"`
		ServerPassword string `json:"server_password"`
		ServerKey      string `json:"server_key"`
		Comment        string `json:"comment"`
	}{}

	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if !u.IsAdmin {
		vars.User = u.User
		vars.Email = u.Email
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	ra, err := orm.NewStmt(db, "cluster").Where("id=%v", vars.ID).Update(&vars)
	if err != nil {
		log.Errorf("add cluster req:%+v, user:%+v error:%s", r, u, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "query db error")
		return
	}

	log.Debugf("user:%v, update cluster:%v, rowsAffect:%v", u, vars, ra)
	server.SendResponseData(w, ra)
}

func (c *cluster) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Name           string `json:"name"`
		User           string `json:"user"`
		RoleID         int64  `json:"role"`
		Email          string `json:"email"`
		ServerUser     string `json:"server_user"`
		ServerPassword string `json:"server_password"`
		ServerKey      string `json:"server_key"`
		Comment        string `json:"comment"`
	}{}

	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if !u.IsAdmin {
		vars.User = u.User
		vars.Email = u.Email
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	id, err := orm.NewStmt(db, "cluster").Insert(&vars)
	if err != nil {
		log.Errorf("add cluster req:%+v, user:%+v error:%s", r, u, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "query db error")
		return
	}

	log.Debugf("user:%v, add cluster:%v, id:%v", u, vars, id)
	server.SendResponseData(w, id)
}

func (c *cluster) DELETE(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id"`
	}{}

	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	sql := fmt.Sprintf("delete from cluster where id = %d and role_id in (%v)", vars.ID, u.RolesKey)
	id, err := orm.NewStmt(db, "cluster").Exec(sql)
	if err != nil {
		log.Errorf("add cluster req:%+v, user:%+v error:%s", r, u, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "query db error")
		return
	}

	log.Debugf("user:%v, add cluster:%v, id:%v", u, vars, id)
	server.SendData(w, id)
}

// clusterInfo 集群基本信息.
type clusterInfo struct {
}

func (i *clusterInfo) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id"`
	}{}

	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	var c cluster
	if err = orm.NewStmt(db, "cluster").Where("id = %d", vars.ID).Query(&c); err != nil {
		log.Errorf("Query cluster req:%+v, user:%+v error:%s", r, u, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "query db error")
		return
	}

	log.Debugf("user:%v, cluster:%v", u, c)
	server.SendResponseData(w, c)
}

// clusterNode 集群节点信息.
type clusterNode struct {
}

func (n *clusterNode) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"cluster_id"`
	}{}

	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	var ns []node
	if err = orm.NewStmt(db, "node").Where("cluster_id = %d", vars.ID).Query(&ns); err != nil {
		log.Errorf("Query node req:%+v, user:%+v error:%s", r, u, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "query db error")
		return
	}

	log.Debugf("user:%v, node:%v", u, ns)
	server.SendData(w, ns)
}

func (n *clusterNode) DELETE(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id"`
	}{}

	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	sql := fmt.Sprintf("delete from node where id=%d and cluster_id in (select id from cluster where role_id in (%v) )", vars.ID, u.RolesKey)
	if _, err = orm.NewStmt(db, "").Exec(sql); err != nil {
		log.Errorf("delete node req:%+v, user:%+v error:%s", r, u, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "query db error")
		return
	}

	log.Debugf("user:%v, delete node:%v", u, vars.ID)
	server.SendResponseOK(w)
}

func (n *clusterNode) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ClusterID int64  `json:"cluster_id"`
		Server    string `json:"server"`
		Comment   string `json:"comment"`
	}{}

	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	id, err := orm.NewStmt(db, "node").Insert(&vars)
	if err != nil {
		log.Errorf("add node req:%+v, user:%+v error:%s", r, u, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "query db error")
		return
	}

	log.Debugf("user:%v, new node:%v, id:%v", u, vars, id)
	server.SendResponseData(w, id)
}
