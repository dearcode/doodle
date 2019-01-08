package manager

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dearcode/crab/log"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/meta"
	"github.com/dearcode/doodle/util"
)

type variableInfo struct {
}

func (vi *variableInfo) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		InterfaceID int64 `json:"interfaceID" valid:"Required"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	where := fmt.Sprintf("interface_id =%d", vars.InterfaceID)

	var rows []meta.Variable

	total, err := query("variable", where, "", "", 0, 0, &rows)
	if err != nil {
		log.Errorf("query err:%s", err.Error())
	}

	if len(rows) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total":0,"rows":[]}`))
		log.Debugf("variable with interfaceID:%d, not found", vars.InterfaceID)
		return
	}

	result := struct {
		Total int             `json:"total"`
		Rows  []meta.Variable `json:"rows"`
	}{total, rows}

	buf, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)

	log.Debugf("infos Variable:%+v\n", string(buf))
}

func (v *variable) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		InterfaceID int64  `json:"interfaceID" valid:"Required"`
		ServiceID   int64  `json:"serviceID" valid:"Required"`
		Sort        string `json:"sort"`
		Order       string `json:"order"`
		Page        int    `json:"offset"`
		Size        int    `json:"limit"`
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

	resID, err := getServiceResourceID(vars.ServiceID)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	if err = u.assert(resID); err != nil {
		log.Errorf("resourceID:%d, vars:%+v, err:%v", resID, vars, errors.ErrorStack(err))
		fmt.Fprintf(w, err.Error())
		return
	}

	where := fmt.Sprintf("interface_id =%d", vars.InterfaceID)

	var rows []meta.Variable

	total, err := query("variable", where, vars.Sort, vars.Order, vars.Page, vars.Size, &rows)
	if err != nil {
		log.Errorf("query err:%s", err.Error())
	}

	if len(rows) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total":0,"rows":[]}`))
		log.Debugf("variable with interfaceID:%d, not found", vars.InterfaceID)
		return
	}

	result := struct {
		Total int             `json:"total"`
		Rows  []meta.Variable `json:"rows"`
	}{total, rows}

	buf, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)

	log.Debugf("Variable:%+v\n", string(buf))
}

type variable struct {
}

func (v *variable) DELETE(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id"`
	}{}

	_, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = del("variable", vars.ID); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponse(w, 0, "")

	log.Debugf("delete Variable:%v, success", vars.ID)
}

func (v *variable) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		InterfaceID int64  `db:"interface_id" json:"interfaceID" valid:"Required"`
		Postion     int    `json:"postion"`
		Name        string `json:"name"  valid:"Required"`
		Type        string `json:"type"  valid:"Required"`
		Required    int    `db:"required" json:"required"`
		Example     string `json:"example"  valid:"Required"`
		Comment     string `json:"comment"  valid:"Required"`
		CTime       string `db_default:"now()"`
		Mtime       string `db_default:"now()"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		log.Errorf("invalid req:%+v", r)
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	id, err := add("variable", vars)
	if err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponse(w, 0, `{"id":%d}`, id)

	log.Debugf("add Variable success, id:%v", id)
}

func (v *variable) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID       int64  `json:"id" valid:"Required"`
		Postion  int    `json:"postion"`
		Name     string `json:"name"  valid:"Required"`
		Type     string `json:"type"  valid:"Required"`
		Required int    `json:"required"`
		Example  string `json:"example"  valid:"Required"`
		Comment  string `json:"comment"  valid:"Required"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		log.Errorf("invalid req:%+v", r)
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := updateVariable(vars.ID, vars.Postion, vars.Name, vars.Type, vars.Required, vars.Example, vars.Comment); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponse(w, 0, "")

	log.Debugf("update Variable success, new:%+v", vars)
}
