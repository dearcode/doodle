package manager

import (
	"encoding/json"
	"fmt"
	"net/http"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/orm"
	"github.com/juju/errors"

	"dearcode.net/doodle/pkg/manager/config"
	"dearcode.net/doodle/pkg/meta"
	"dearcode.net/doodle/pkg/util"
	"dearcode.net/doodle/pkg/util/etcd"
)

type nodes struct {
}

const (
	etcdAPIPrefix = "/api"
)

func (n *nodes) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ServiceID int64 `json:"serviceID" valid:"Required"`
	}{}

	if err := server.ParseURLVars(r, &vars); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection error:%v", errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	var p meta.Service

	if err = orm.NewStmt(db, "service").Where("id=%d", vars.ServiceID).Query(&p); err != nil {
		log.Errorf("query service:%d error:%v", vars.ServiceID, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	key := etcdAPIPrefix + p.Source[6:]
	e, err := etcd.New(config.Manager.ETCD.Hosts)
	if err != nil {
		log.Errorf("connect etcd:%v error:%v", config.Manager.ETCD.Hosts, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	km, err := e.List(key)
	if err != nil {
		log.Errorf("list etcd key:%v error:%v", key, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	var rows []meta.MicroAPP

	for _, v := range km {
		var a meta.MicroAPP
		json.Unmarshal([]byte(v), &a)
		rows = append(rows, a)
	}

	log.Debugf("service:%v nodes:%+v", vars.ServiceID, rows)
	server.SendData(w, rows)
}
