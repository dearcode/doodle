package manager

import (
	"fmt"
	"net/http"

	"github.com/dearcode/crab/log"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/util"
)

type statsSumAction struct {
	ID int64 `json:"interfaceID"`
}

//GET 查询流量总数
func (ssa *statsSumAction) GET(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, ssa); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	ss, err := selectStats(ssa.ID)
	if err != nil {
		util.SendResponse(w, http.StatusNotFound, "not found")
		log.Errorf("stats not found, error:%v", errors.ErrorStack(err))
		return
	}

	log.Debugf("result:%v", ss)
	response(w, ss)
}

type statsTopInterface struct {
}

// GET 查询流量总数
func (sti *statsTopInterface) GET(w http.ResponseWriter, r *http.Request) {
	tis, err := selectTopIface()
	if err != nil {
		util.SendResponse(w, http.StatusNotFound, "not found")
		log.Errorf("stats not found, error:%v", errors.ErrorStack(err))
		return
	}

	log.Debugf("result:%v", tis)
	response(w, QueryResponse{Total: len(tis), Rows: tis})
}

type statsTopApplication struct {
	ID int64 `json:"interfaceID"`
}

// GET 查询流量总数
func (sta *statsTopApplication) GET(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, sta); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	as, err := selectTopApp(sta.ID)
	if err != nil {
		util.SendResponse(w, http.StatusNotFound, "not found")
		log.Debugf("stats not found")
		return
	}

	log.Debugf("result:%v", as)
	response(w, QueryResponse{Total: len(as), Rows: as})
}

type statsErrors struct {
	ID    int64  `json:"interfaceID"`
	Sort  string `json:"sort"`
	Order string `json:"order"`
	Page  int    `json:"offset"`
	Size  int    `json:"limit"`
}

// GET 查询流量总数
func (se *statsErrors) GET(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, se); err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	var errs []statsError

	where := "stats_error.app_id = application.id and stats_error.iface_id = interface.id and interface.service_id = service.id"
	if se.ID != 0 {
		where = fmt.Sprintf(" stats_error.iface_id = %d and %s", se.ID, where)
	}

	total, err := query("stats_error,application,interface,service", where, se.Sort, se.Order, se.Page, se.Size, &errs)
	if err != nil {
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Debugf("result:%v", errs)
	response(w, QueryResponse{Total: total, Rows: errs})
}
