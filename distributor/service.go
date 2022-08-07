package distributor

import (
	"fmt"
	"net/http"
	"time"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/orm"
	"dearcode.net/crab/util/aes"

	"dearcode.net/doodle/distributor/config"
	"dearcode.net/doodle/util/etcd"
)

type service struct {
	ID      int64
	Source  string
	Name    string
	Cluster cluster `db_table:"one"`
	Ctime   string  `db_default:"now()"`
}

// GET 获取service的部署情况.
func (p *service) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64
	}{}

	if err := server.ParseURLVars(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("connect db error:%v", err.Error())
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	if err = orm.NewStmt(db, "service").Where("id=%v", vars.ID).Query(p); err != nil {
		log.Errorf("query service:%v error:%v", vars.ID, err.Error())
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("service:%+v, id:%v", p, vars.ID)

	c, err := etcd.New(config.Distributor.ETCD.Hosts)
	if err != nil {
		log.Errorf("etcd connect:%s error:%v", config.Distributor.ETCD.Hosts, err.Error())
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer c.Close()

	prefix := fmt.Sprintf("/api%s", p.Name)
	keys, err := c.List(prefix)
	if err != nil {
		log.Errorf("etcd list:%s error:%v", prefix, err.Error())
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Debugf("keys:%v, prefix:%v", keys, prefix)

	server.SendResponseData(w, keys)
}

func (p *service) key() string {
	s := fmt.Sprintf("%x.%v", p.ID, time.Now().UnixNano())
	ns, err := aes.Encrypt(s, config.Distributor.Server.SecretKey)
	if err != nil {
		panic(err.Error())
	}
	return ns
}
