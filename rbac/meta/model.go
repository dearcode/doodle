package meta

/*
import (
	"encoding/json"
	"fmt"
	"net/http"
)
*/

// Role 对应角色表
type Role struct {
	ID       int64  `db:"id" db_default:"auto"`
	AppID    int64  `db:"app_id"`
	Name     string `db:"name"`
	UserID   int64  `db:"user_id"`
	Comments string `db:"comments"`
	Ctime    string `db:"ctime" db_default:"now()"`
	Mtime    string `db:"mtime" db_default:"now()"`
}

// App 应用表
type App struct {
	ID       int64  `db:"id" db_default:"auto"`
	Name     string `db:"name"`
	Email    string `db:"email"`
	Token    string `db:"token" db_default:"''"`
	Comments string `db:"comments"`
	Ctime    string `db:"ctime" db_default:"now()"`
	Mtime    string `db:"mtime" db_default:"now()"`
}

// Resource  资源表
type Resource struct {
	ID       int64  `db:"id" db_default:"auto"`
	AppID    int64  `db:"app_id"`
	Name     string `db:"name"`
	Comments string `db:"comments"`
	Ctime    string `db:"ctime" db_default:"now()"`
	Mtime    string `db:"mtime" db_default:"now()"`
}

//User 用户表.
type User struct {
	ID    int64  `db:"id" db_default:"auto"`
	AppID int64  `db:"app_id"`
	Name  string `db:"name"`
	Email string `db:"email"`
	Ctime string `db:"ctime" db_default:"now()"`
	Mtime string `db:"mtime" db_default:"now()"`
}

//UserInfo 用户信息.
type UserInfo struct {
	ID    int64      `db:"id" db_default:"auto"`
	AppID int64      `db:"app_id"`
	Name  string     `db:"name"`
	Email string     `db:"email"`
	Res   []Resource `db_table:"one2more"`
	Roles []Role     `db_table:"one2more"`
	Ctime string     `db:"ctime" db_default:"now()"`
	Mtime string     `db:"mtime" db_default:"now()"`
}

//RoleResource 角色与资源关联表.
type RoleResource struct {
	ID           int64  `db:"id" db_default:"auto"`
	AppID        int64  `db:"app_id"`
	ResourceID   int64  `db:"resource_id"`
	ResourceName string `db:"resource.name" db_default:""`
	RoleID       int64  `db:"role_id"`
	RoleName     string `db:"role.name" db_default:""`
	RoleComments string `db:"role.comments" db_default:""`
	Ctime        string `db:"ctime" db_default:"now()"`
	Mtime        string `db:"mtime" db_default:"now()"`
}

//RoleUser 角色与用户关联表.
type RoleUser struct {
	ID           int64  `db:"id" db_default:"auto"`
	AppID        int64  `db:"app_id"`
	AdminID      int64  `db:"role.user_id"`
	UserID       int64  `db:"user_id"`
	UserName     string `db:"user.name"`
	UserEmail    string `db:"user.email"`
	RoleID       int64  `db:"role_id"`
	RoleName     string `db:"role.name"`
	RoleComments string `db:"role.comments"`
	Ctime        string `db:"ctime" db_default:"now()"`
	Mtime        string `db:"mtime" db_default:"now()"`
}

/*
//QueryResponse bootstrap格式数据.
type QueryResponse struct {
	Total int         `json:"total"`
	Rows  interface{} `json:"rows"`
}

//Response 标准接口返回.
type Response struct {
	Status  int
	Message string      `json:",omitempty"`
	Data    interface{} `json:",omitempty"`
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
*/
