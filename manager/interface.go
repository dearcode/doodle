package manager

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/dearcode/crab/http/server"
	"github.com/dearcode/crab/log"
	"github.com/dearcode/crab/orm"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/meta"
	"github.com/dearcode/doodle/meta/document"
	"github.com/dearcode/doodle/util"
)

type interfaceRun struct {
}

func (i *interfaceRun) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id" valid:"Required"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	iface, err := queryInterfaceInfo(vars.ID)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	var vs []meta.Variable

	if _, err = query("variable", fmt.Sprintf("interface_id=%d", vars.ID), "id", "asc", 0, 0, &vs); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
	log.Debugf("load var id:%d data:%+v", vars.ID, vs)

	req := &http.Request{}
	backend := iface.Backend
	if !strings.Contains(backend, "?") {
		backend += fmt.Sprintf("?t=%d", time.Now().Unix())
	}
	req.Header = make(map[string][]string)

	reqBody := bytes.NewBuffer([]byte{})

	for _, v := range vs {
		val := r.FormValue(v.Name)
		if val == "" && v.Required {
			fmt.Fprintf(w, "字段:"+v.Name+"不能为空")
			return
		}
		switch v.Postion {
		case server.URI:
			backend += fmt.Sprintf("&%s=%s", v.Name, val)
		case server.HEADER:
			if req.Header == nil {
				req.Header = make(map[string][]string)
			}
			req.Header.Add(v.Name, val)
			log.Debugf("header add name:%v, val:%v", v.Name, val)
		case server.FORM:
			if reqBody.Len() > 0 {
				reqBody.WriteString("&")
			}
			reqBody.WriteString(fmt.Sprintf("%s=%s", v.Name, url.QueryEscape(val)))
		}
	}

	if req.URL, err = req.URL.Parse(backend); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	if iface.Method == server.GET {
		req.Method = "GET"
	} else {
		req.Method = "POST"
	}

	req.ContentLength = int64(reqBody.Len())
	req.Body = ioutil.NopCloser(reqBody)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	c := &http.Client{Timeout: time.Second * 5}
	log.Debugf("test req:%+v", req)
	resp, err := c.Do(req)
	if err != nil {
		log.Debugf("do error:%v", err.Error())
		fmt.Fprintf(w, err.Error())
		return
	}

	buf, err := httputil.DumpResponse(resp, true)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	//w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

type interfaceInfo struct {
}

//GET interfaceInfo.
func (ii *interfaceInfo) GET(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id" valid:"Required"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	iface, err := queryInterfaceInfo(vars.ID)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
	iface.ID = vars.ID

	buf, err := json.Marshal(iface)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)

	log.Debugf("find id:%d, interface:%+v\n", vars.ID, iface)
}

type interfaceAction struct {
	ID        int64  `json:"id"`
	State     int    `json:"state"`
	ServiceID int64  `json:"pid"`
	Sort      string `json:"sort"`
	Order     string `json:"order"`
	Page      int    `json:"offset"`
	Size      int    `json:"limit"`
	Search    string `json:"search"`
}

func (i *interfaceAction) GET(w http.ResponseWriter, r *http.Request) {
	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, i); err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	resID, err := getServiceResourceID(i.ServiceID)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	if err = u.assert(resID); err != nil {
		log.Errorf("resourceID:%d, vars:%+v, err:%v", resID, i, errors.ErrorStack(err))
		fmt.Fprintf(w, err.Error())
		return
	}

	var is []meta.Interface

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("resourceID:%d, vars:%+v, err:%v", resID, i, errors.ErrorStack(err))
		fmt.Fprintf(w, err.Error())
		return
	}

	stmt := orm.NewStmt(db, "interface")
	stmt = stmt.Where("service_id=%d", i.ServiceID)

	if i.State == 1 {
		stmt = stmt.Where("state=1")
	}

	if i.Search != "" {
		stmt = stmt.Where("(interface.name like '%" + i.Search + "%'" +
			" or interface.user like '%" + i.Search + "%'" +
			" or interface.comment like '%" + i.Search + "%'" +
			" or interface.path like '%" + i.Search + "%'" +
			" or interface.backend like '%" + i.Search + "%')")
	}

	total, err := stmt.Count()
	if err != nil {
		log.Errorf("count interface error:%v", errors.ErrorStack(err))
		fmt.Fprintf(w, err.Error())
		return
	}
	log.Debugf("count:%d", total)

	if i.Sort != "" {
		stmt = stmt.Order("interface." + i.Sort)
	}

	stmt = stmt.Offset(i.Page).Limit(i.Size)
	if err = stmt.Query(&is); err != nil {
		log.Errorf("query interface:%+v error:%v", i, errors.ErrorStack(err))
		server.SendRows(w, 0, nil)
		return
	}

	server.SendRows(w, total, is)
}

func (i *interfaceAction) DELETE(w http.ResponseWriter, r *http.Request) {
	if err := util.DecodeRequestValue(r, i); err != nil {
		log.Errorf("%v DecodeRequestValue error:%v", r.RemoteAddr, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := del("interface", i.ID); err != nil {
		log.Errorf("%v interface error:%v", i, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	util.SendResponse(w, 0, "")

	log.Debugf("delete Interface:%v, success", i.ID)
}

func (i *interfaceAction) POST(w http.ResponseWriter, r *http.Request) {
	vars := iface{}

	u, err := session.User(w, r)
	if err != nil {
		log.Errorf("session.User error:%v, req:%v", errors.ErrorStack(err), r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = util.DecodeRequestValue(r, &vars); err != nil {
		log.Errorf("invalid req:%+v", r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	resID, err := getServiceResourceID(vars.ServiceID)
	if err != nil {
		log.Errorf("invalid req:%+v", r)
		util.SendResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err = u.assert(resID); err != nil {
		log.Errorf("resourceID:%d, vars:%+v, err:%v", resID, vars, errors.ErrorStack(err))
		fmt.Fprintf(w, err.Error())
		return
	}

	vars.State = 0
	vars.User = u.User
	vars.Email = u.Email

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("GetConnection error:%v", errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	id, err := orm.NewStmt(db, "interface").Insert(vars)
	if err != nil {
		if strings.Contains(err.Error(), "1062") {
			log.Errorf("add req:%+v, error:%s", r, errors.ErrorStack(err))
			util.SendResponse(w, http.StatusInternalServerError, "接口路径已存在, 接口路径在整个项目中是唯一的，不可重复")
			return
		}
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponseJSON(w, &id)

	log.Debugf("add Interface success, id:%v", id)
}

func (i *interfaceAction) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID      int64  `json:"id" valid:"Required"`
		Name    string `json:"name"  valid:"Required"`
		User    string `json:"user"`
		Email   string `json:"email"`
		Method  int    `json:"method"`
		Path    string `json:"path"  valid:"AlphaNumeric"`
		Backend string `json:"backend"  valid:"Required"`
		Comment string `json:"comment"  valid:"Required"`
		Level   int    `json:"level"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		log.Errorf("invalid req:%+v", r)
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := updateInterface(vars.ID, vars.Method, vars.Level, vars.Name, vars.Path, vars.Backend, vars.Comment, vars.User, vars.Email); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponse(w, 0, "")

	log.Debugf("update Interface success, new:%+v", vars)
}

type interfaceDeploy struct {
}

func (id *interfaceDeploy) PUT(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ID int64 `json:"id" valid:"Required"`
	}{}

	if err := util.DecodeRequestValue(r, &vars); err != nil {
		log.Errorf("invalid req:%+v", r)
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := deployInterface(vars.ID); err != nil {
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.SendResponse(w, 0, "")

	log.Debugf("deploy Interface:%d success", vars.ID)
}

type interfaceRegister struct {
}

type jsonVariable struct {
	InterfaceID int64
	Postion     int
	Name        string
	Level       int
	Parent      string
	Type        string
	Required    bool
	Example     string
	Comment     string
}

func (ir *interfaceRegister) addVariable(level int, parent string, db *sql.DB, interfaceID int64, postion int, fields map[string]document.Field) error {
	vars := jsonVariable{
		InterfaceID: interfaceID,
		Postion:     postion,
		Level:       level,
		Parent:      parent,
	}

	for _, v := range fields {
		vars.Name = v.Name
		vars.Comment = v.Comment
		vars.Type = v.Type
		vars.Required = v.Required

		id, err := orm.NewStmt(db, "variable").Insert(&vars)
		if err != nil {
			log.Errorf("insert variable:%+v, err:%v", vars, errors.ErrorStack(err))
			return errors.Trace(err)
		}

		log.Debugf("new variable:%+v id:%v", vars, id)

		if v.Child != nil {
			if err = ir.addVariable(level+1, vars.Type, db, interfaceID, postion, v.Child); err != nil {
				return errors.Trace(err)
			}
		}
	}

	return nil
}

const (
	postionRequestJSON  = 4
	postionResponseJSON = 14
)

func (ir *interfaceRegister) POST(w http.ResponseWriter, r *http.Request) {
	vars := struct {
		ServiceID int64
		User      string
		State     int
		Email     string
		Name      string
		Version   int64
		Path      string
		Method    int
		Backend   string
		Comment   string
		Attr      document.Method `db_default:"auto"`
	}{
		State: 1,
	}

	if err := server.ParseJSONVars(r, &vars); err != nil {
		log.Errorf("invalid req:%+v", r)
		server.SendResponse(w, http.StatusBadRequest, err.Error())
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

	if err = orm.NewStmt(db, "service").Where("id=%v", vars.ServiceID).Query(&p); err != nil {
		log.Errorf("Query service:%v error:%v", vars.ServiceID, errors.ErrorStack(err))
		util.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if vars.Email == "" {
		vars.User = p.User
		vars.Email = p.Email
	}

	id, err := orm.NewStmt(db, "interface").Insert(&vars)
	if err != nil {
		if !strings.Contains(err.Error(), "Error 1062: Duplicate entry") {
			log.Errorf("insert interface:%+v, error:%v", vars, err)
			server.SendResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		var i meta.Interface
		if err = orm.NewStmt(db, "interface").Where("service_id=%d and interface.path='%s' and method=%d", vars.ServiceID, vars.Path, vars.Method).Query(&i); err != nil {
			log.Errorf("query interface:%+v, error:%v", vars, err)
			server.SendResponse(w, http.StatusInternalServerError, err.Error())
			return

		}

		id = i.ID

		if _, err = orm.NewStmt(db, "interface").Where("id=%d", id).Update(&vars); err != nil {
			log.Errorf("interface:%+v, error:%v", vars, err)
			server.SendResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		if _, err = orm.NewStmt(db, "variable").Exec("delete from variable where interface_id=?", id); err != nil {
			log.Errorf("clear variable:%+v, error:%v", vars, err)
			server.SendResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err = ir.addVariable(0, "", db, id, postionRequestJSON, vars.Attr.Request); err != nil {
		log.Errorf("add request variable:%+v, error:%v", vars.Attr.Request, err)
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err = ir.addVariable(0, "", db, id, postionResponseJSON, vars.Attr.Response); err != nil {
		log.Errorf("add response variable:%+v, error:%v", vars.Attr.Response, err)
		server.SendResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.SendResponseData(w, id)
	log.Debugf("new interface:%+v, id:%v", vars, id)
}
