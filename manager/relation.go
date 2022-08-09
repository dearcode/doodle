package manager

import (
	"encoding/json"
	"fmt"
	"net/http"

	"dearcode.net/crab/log"
	"github.com/juju/errors"

	"dearcode.net/doodle/meta"
	"dearcode.net/doodle/util"
)

type relation struct {
}

func (ra *relation) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		AppID       int64  `json:"applicationID"`
		InterfaceID int64  `json:"interfaceID"`
		Sort        string `json:"sort"`
		Order       string `json:"order"`
		Page        int    `json:"offset"`
		Size        int    `json:"limit"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	var where string

	switch vars.Sort {
	case "InterfaceName":
		vars.Sort = "interface.name"
	case "ServiceName":
		vars.Sort = "service.name"
	}

	if vars.AppID != 0 {
		where = fmt.Sprintf("relation.application_id=%d", vars.AppID)
	}

	if vars.InterfaceID != 0 {
		if where != "" {
			where += " and "
		}
		where += fmt.Sprintf("relation.interface_id=%d", vars.InterfaceID)
	}

	if where != "" {
		where += " and "
	}
	where += " relation.interface_id=interface.id and relation.application_id = application.id and  interface.service_id = service.id"

	var rs []meta.Relation

	total, err := query("relation, application, interface, service", where, vars.Sort, vars.Order, vars.Page, vars.Size, &rs)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	if len(rs) == 0 {
		w.Header().Set("Content-Type", "Relationlication/json")
		w.Write([]byte(`{"total":0,"rows":[]}`))
		log.Debugf("service not found")
		return
	}

	result := struct {
		Total int             `json:"total"`
		Rows  []meta.Relation `json:"rows"`
	}{total, rs}

	buf, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "Relationlication/json")
	w.Write(buf)
}

func (ra *relation) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID          int64 `json:"id"`
		AppID       int64 `json:"appID"`
		InterfaceID int64 `json:"interfaceID"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		log.Errorf("DecodeRequestValue req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := updateRelation(vars.ID, vars.InterfaceID, vars.AppID); err != nil {
		log.Errorf("updateRelation req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	util.SendResponse(w, 0, "")

	log.Debugf("update relation success, vars:%+v", vars)
}

// TODO 删除应用，删除接口时要调用这个接口把关系也删除了
func (ra *relation) DELETE(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := del("relation", vars.ID); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponse(w, 0, "")

	log.Debugf("delete service:%v, success", vars.ID)
}

func (ra *relation) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		AppID       int64  `db:"application_id" json:"appID"`
		InterfaceID int64  `db:"interface_id" json:"interfaceID"`
		CTime       string `db_default:"now()"`
		Mtime       string `db_default:"now()"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		log.Errorf("DecodeRequestValue req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	state, err := getInterfaceState(vars.InterfaceID)
	if err != nil {
		log.Errorf("getInterfaceState req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if state != 1 {
		log.Errorf("interface:%d state:%d", vars.InterfaceID, state)
		util.SendResponse(w, http.StatusInternalServerError, "接口未发布，不可以授权应用")
		return
	}

	id, err := add("relation", vars)
	if err != nil {
		log.Errorf("addRelation req:%+v, error:%s", r, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	util.SendResponse(w, 0, fmt.Sprintf(`{"id":%d}`, id))

	log.Debugf("add relation success, id:%v, %+v", id, vars)
}
