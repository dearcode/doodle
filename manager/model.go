package manager

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// userinfo erp中用户信息
type userinfo struct {
	Status   int
	IsAdmin  bool
	Res      []int64
	ResKey   string
	Roles    []int64
	RolesKey string
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	User     string `json:"fullname"`
	UserID   int64  `json:"userId"`
}

type statsSum struct {
	Date string
	Sum  int64
	Avg  int64
}

type statsTopApp struct {
	AppID         int64
	AppName       string
	AppUser       string
	InterfaceID   int64
	InterfaceName string
	InterfaceUser string
	ServiceID     int64
	ServiceName   string
	Value         int64
}

type statsTopIface struct {
	ID            int64  `json:"id"`
	ServiceName   string `json:"service"`
	InterfaceName string `json:"iface"`
	User          string `json:"user"`
	Value         int64  `json:"value"`
}

// QueryResponse 专门给bootstrap-table用的.
type QueryResponse struct {
	Total int         `json:"total"`
	Rows  interface{} `json:"rows"`
}

func response(w http.ResponseWriter, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	buf, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(w, `{"Status":500, "Message":"%s"}`, err.Error())
		return
	}
	if _, err = w.Write(buf); err != nil {
		fmt.Fprintf(w, `{"Status":500, "Message":"%s"}`, err.Error())
	}
}

type iface struct {
	ID        int64  `db_default:"auto"`
	ServiceID int64  `json:"pid" valid:"Required"`
	Name      string `json:"name"  valid:"Required"`
	Method    int    `json:"method"`
	User      string `json:"user"`
	Email     string `json:"email"`
	State     int
	Path      string `json:"path"  valid:"AlphaNumeric"`
	Backend   string `json:"backend"  valid:"Required"`
	Comment   string `json:"comment"  valid:"Required"`
	Level     int    `json:"level"`
	CTime     string `db_default:"now()"`
	Mtime     string `db_default:"now()"`
}

type statsError struct {
	ID          int64
	Session     string
	AppID       int64  `db:"app_id"`
	AppName     string `db:"application.name"`
	IfaceID     int64  `db:"iface_id"`
	IfaceName   string `db:"interface.name"`
	ServiceName string `db:"service.name"`
	Info        string
	Ctime       string
}
