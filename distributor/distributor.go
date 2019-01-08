package distributor

import (
	"net/http"

	"github.com/dearcode/crab/http/server"
	"github.com/dearcode/crab/log"
	"github.com/juju/errors"
)

type distributorLogs struct {
	ID            int64
	DistributorID int64
	State         int
	PID           int
	INFO          string
	CreateTime    string `db_default:"now()"`
}

type distributor struct {
	ID         int64
	ProjectID  int64
	Project    project `db_table:"one"`
	State      int
	Server     string
	CreateTime string `db_default:"now()"`
}

//GET 编译并更新指定项目.
func (d *distributor) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ProjectID int64 `json:"id"`
	}{}

	if err := server.ParseURLVars(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	t, err := newTask(vars.ProjectID)
	if err != nil {
		log.Errorf("newWorkspace error:%v", errors.ErrorStack(err))
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Debugf("newWorkspace:%+v", t.ID)

	server.SendResponseData(w, t.d.ID)

	go d.run(t)
}

//POST 编译并更新指定项目.
func (d *distributor) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ProjectID int64 `json:"id"`
	}{}

	if err := server.ParseURLVars(r, &vars); err != nil {
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	t, err := newTask(vars.ProjectID)
	if err != nil {
		log.Errorf("newWorkspace error:%v", errors.ErrorStack(err))
		server.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Debugf("newWorkspace:%+v", t.ID)

	server.SendResponseData(w, t.d.ID)

	go d.run(t)
}

func (d *distributor) run(t *task) {
	if err := t.compile(); err != nil {
		log.Errorf("newWorkspace error:%v", errors.ErrorStack(err))
		return
	}

	if err := t.install(); err != nil {
		log.Errorf("newWorkspace error:%v", errors.ErrorStack(err))
		return
	}

}
