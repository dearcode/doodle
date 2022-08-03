package manager

import (
	"fmt"
	"net/http"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/orm"
	"github.com/juju/errors"

	"dearcode.net/doodle/meta"
	"dearcode.net/doodle/util"
)

type docs struct {
}

// GET get docs.
func (d *docs) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ServiceName   string `json:"serviceName"`
		InterfaceName string `json:"interfaceName"`
		Sort          string `json:"sort"`
		Order         string `json:"order"`
		Page          int    `json:"offset"`
		Size          int    `json:"limit"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	where := "interface.state = 1"

	if vars.ServiceName != "" {
		if where != "" {
			where += " and "
		}
		where += fmt.Sprintf("service.name like '%%%s%%'", vars.ServiceName)
	}

	if vars.InterfaceName != "" {
		if where != "" {
			where += " and "
		}
		where += fmt.Sprintf("interface.name like '%%%s%%'", vars.InterfaceName)
	}

	switch vars.Sort {
	case "InterfaceName":
		vars.Sort = "interface.name"
	case "ServiceName":
		vars.Sort = "service.name"
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, "连接数据库出错")
		return
	}
	defer db.Close()

	var is []meta.Interface

	stmt := orm.NewStmt(db, "interface").Where(where)

	total, err := stmt.Count()
	if err != nil {
		log.Errorf("query interface,service count error:%v, vars:%v", errors.ErrorStack(err), vars)
		fmt.Fprintf(w, err.Error())
		return
	}
	if total == 0 {
		server.SendRows(w, 0, nil)
		return
	}

	if err = stmt.Order(vars.Order).Sort(vars.Sort).Offset(vars.Page).Limit(vars.Size).Query(&is); err != nil {
		log.Errorf("query interface,service error:%v, vars:%v", err, vars)
		fmt.Fprintf(w, err.Error())
		return
	}

	server.SendRows(w, total, is)
}
