package rbac

import (
	"net/http"

	"github.com/dearcode/crab/http/server"
	"github.com/dearcode/crab/log"
	"github.com/dearcode/crab/orm"
)

type authorize struct {
	Name     string
	Password string
	Salt     string
	Callback string
	Error    string
}

func (a authorize) GET(w http.ResponseWriter, r *http.Request) {
	if err := server.ParseURLVars(r, &a); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	execute(w, a)
}

func (a authorize) POST(w http.ResponseWriter, r *http.Request) {
	if err := server.ParseFormVars(r, &a); err != nil {
		log.Errorf("parse form error")
		server.Abort(w, "parse form error")
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("connect db error")
		server.Abort(w, "connect db error")
		return
	}
	defer db.Close()

	var ac account

	if err = orm.NewStmt(db, "account").SetLogger(log.GetLogger()).
		Where("name='%v' and md5(concat(password, '%s')) = '%s'", a.Name, a.Salt, a.Password).
		Query(&ac); err != nil {
		log.Errorf("query account error:%v", err)
		server.Abort(w, "%s", err.Error())
		return
	}

	token := ac.token()

	log.Debugf("user:%s, token:%s", a.Name, token)
	server.SendResponseData(w, token)
}
