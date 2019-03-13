package rbac

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/dearcode/crab/http/server"
	"github.com/dearcode/crab/log"
	"github.com/dearcode/crab/orm"
	"github.com/dearcode/crab/util/aes"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/rbac/config"
)

type account struct {
	ID    int64
	Name  string
	Email string
}

//token 直接返回urlencode之后的数据，方便调试
func (a account) token() string {
	s := fmt.Sprintf("%d.%d", a.ID, time.Now().Unix())
	nk, err := aes.Encrypt(s, config.RBAC.Server.Key)
	if err != nil {
		panic(err.Error())
	}

	return url.QueryEscape(nk)
}

func (a account) parseToken(token string) (int64, time.Time, error) {
	s, err := aes.Decrypt(token, config.RBAC.Server.Key)
	if err != nil {
		return 0, time.Time{}, errors.Trace(err)
	}

	var id, sec int64

	if _, err := fmt.Sscanf(string(s), "%d.%d", &id, &sec); err != nil {
		return 0, time.Time{}, errors.Annotatef(err, "%s", s)
	}

	return id, time.Unix(sec, 0), nil
}

func (a *account) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		Token string `json:"token"`
	}{}

	if err := server.ParseURLVars(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("connect db error:%v", errors.ErrorStack(err))
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	defer db.Close()

	id, t, err := a.parseToken(vars.Token)
	if err != nil {
		log.Errorf("parse token error:%v, token:%v", errors.ErrorStack(err), vars.Token)
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	//5秒之前的不认.
	if t.Before(time.Now().Add(-time.Second * 5)) {
		log.Errorf("token timeout, token:%v, time:%v, now:%v", vars.Token, t, time.Now())
		server.SendResponse(w, http.StatusBadRequest, "token timeout")
		return
	}

	var ac account

	if err = orm.NewStmt(db, "account").Where("id=%d", id).Query(&ac); err != nil {
		log.Errorf("query account error:%v, id:%v, time:%v", err, id, t)
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("token:%v, account:%v", vars.Token, ac)

	server.SendResponseData(w, ac)
}
