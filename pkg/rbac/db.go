package rbac

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"dearcode.net/crab/log"
	"github.com/juju/errors"

	"dearcode.net/doodle/pkg/rbac/meta"
)

// queryAll 返回所有结果(不分页), result 必需是一个指向切片的指针
func queryAll(table, where string, result interface{}) error {
	rt := reflect.TypeOf(result)
	rv := reflect.ValueOf(result).Elem()

	if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("result type must be ptr to slice, recv:%v", rt.Kind())
	}

	fs := rt.Elem().Elem()
	if fs.NumField() == 0 {
		return fmt.Errorf("result not found field")
	}

	dt := strings.Split(table, ",")[0]

	fields := bytes.NewBuffer([]byte{})
	for i := 0; i < fs.NumField(); i++ {
		name := fs.Field(i).Tag.Get("db")
		if name == "" {
			name = strings.ToLower(fs.Field(i).Name)
		}
		if !strings.Contains(name, ".") {
			fields.WriteString(dt)
			fields.WriteString(".")
		}
		fields.WriteString(name)
		fields.WriteString(", ")
	}

	fields.Truncate(fields.Len() - 2)

	bs := bytes.NewBufferString("select ")
	bs.WriteString(fields.String())
	bs.WriteString(" from ")
	bs.WriteString(table)

	if where != "" {
		bs.WriteString(" where ")
		bs.WriteString(where)
	}

	sql := bs.String()
	log.Debugf("sql:%v", sql)

	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()

	rows, err := db.Query(sql)
	if err != nil {
		return errors.Trace(err)
	}
	defer rows.Close()

	for rows.Next() {
		var refs []interface{}
		obj := reflect.New(fs)

		for i := 0; i < obj.Elem().NumField(); i++ {
			refs = append(refs, obj.Elem().Field(i).Addr().Interface())
		}

		if err := rows.Scan(refs...); err != nil {
			return errors.Trace(err)
		}
		rv = reflect.Append(rv, obj.Elem())
	}

	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(rv.Interface()))

	log.Debugf("result %v", result)
	return nil
}

// result 必需是一个指向切片的指针
func query(table, where, sort, order string, offset, count int, result interface{}) (int, error) {
	rt := reflect.TypeOf(result)
	rv := reflect.ValueOf(result).Elem()

	if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Slice {
		return 0, fmt.Errorf("result type must be ptr to slice, recv:%v", rt.Kind())
	}

	fs := rt.Elem().Elem()
	if fs.NumField() == 0 {
		return 0, fmt.Errorf("result not found field")
	}

	dt := strings.Split(table, ",")[0]

	fields := bytes.NewBuffer([]byte{})
	for i := 0; i < fs.NumField(); i++ {
		name := fs.Field(i).Tag.Get("db")
		if name == "" {
			name = strings.ToLower(fs.Field(i).Name)
		}
		if !strings.Contains(name, ".") {
			fields.WriteString(dt)
			fields.WriteString(".")
		}
		fields.WriteString(name)
		fields.WriteString(", ")
	}

	fields.Truncate(fields.Len() - 2)

	bs := bytes.NewBufferString("select ")
	bs.WriteString(fields.String())
	bs.WriteString(" from ")
	bs.WriteString(table)

	bc := bytes.NewBufferString("select count(*) from ")
	bc.WriteString(table)

	if where != "" {
		bs.WriteString(" where ")
		bs.WriteString(where)

		bc.WriteString(" where ")
		bc.WriteString(where)
	}

	c := bc.String()
	log.Debugf("sql count:%v", c)

	if sort != "" {
		bs.WriteString(" order by ")
		bs.WriteString(sort)
		if order != "" {
			bs.WriteString(" ")
			bs.WriteString(order)
		}
	}

	if count > 0 {
		bs.WriteString(fmt.Sprintf(" limit %d,%d", offset, count))
	}

	sql := bs.String()
	log.Debugf("sql:%v", sql)

	db, err := mdb.GetConnection()
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer db.Close()

	rows, err := db.Query(sql)
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer rows.Close()

	for rows.Next() {
		var refs []interface{}
		obj := reflect.New(fs)

		for i := 0; i < obj.Elem().NumField(); i++ {
			refs = append(refs, obj.Elem().Field(i).Addr().Interface())
		}

		if err := rows.Scan(refs...); err != nil {
			return 0, errors.Trace(err)
		}
		rv = reflect.Append(rv, obj.Elem())
	}

	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(rv.Interface()))

	// select count
	row := db.QueryRow(c)
	row.Scan(&count)

	log.Debugf("result total:%d:%v", count, result)
	return count, nil
}

func add(table string, data interface{}) (int64, error) {
	rt := reflect.TypeOf(data)
	rv := reflect.ValueOf(data)

	if rt.NumField() == 0 {
		return 0, fmt.Errorf("data not found field")
	}

	bs := bytes.NewBufferString("insert into ")
	bs.WriteString(table)
	bs.WriteString(" (")

	for i := 0; i < rt.NumField(); i++ {
		//跳过自增变量
		if rt.Field(i).Tag.Get("db_default") == "auto" {
			continue
		}
		name := rt.Field(i).Tag.Get("db")
		if strings.Contains(name, ".") {
			continue
		}

		if name == "" {
			name = rt.Field(i).Name
		}
		bs.WriteString(name)
		bs.WriteString(", ")
	}
	bs.Truncate(bs.Len() - 2)

	bs.WriteString(") values (")
	for i := 0; i < rt.NumField(); i++ {
		//跳过自增变量
		if rt.Field(i).Tag.Get("db_default") == "auto" {
			continue
		}
		if strings.Contains(rt.Field(i).Tag.Get("db"), ".") {
			continue
		}
		switch rt.Field(i).Type.Kind() {
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			bs.WriteString(fmt.Sprintf("%d, ", rv.Field(i).Int()))
		case reflect.Bool:
			if rv.Field(i).Bool() {
				bs.WriteString("1, ")
			} else {
				bs.WriteString("0, ")
			}
		case reflect.String:
			if rv.Field(i).String() == "" {
				bs.WriteString(rt.Field(i).Tag.Get("db_default") + ", ")
			} else {
				bs.WriteString("'" + rv.Field(i).String() + "', ")
			}
		}
	}
	bs.Truncate(bs.Len() - 2)
	bs.WriteString(") ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id), mtime=now()")

	sql := bs.String()
	log.Debugf("sql:%v", sql)
	db, err := mdb.GetConnection()
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer db.Close()
	r, err := db.Exec(sql)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return r.LastInsertId()
}

func updateAppToken(id int64, token string) error {
	sql := "update app set token=?, mtime=now() where id=?"
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()

	_, err = db.Exec(sql, token, id)
	return errors.Trace(err)
}

func updateRole(role meta.Role) error {
	sql := fmt.Sprintf("update role set name='%s', comments='%s', mtime=now() where id=%d and app_id=%d", role.Name, role.Comments, role.ID, role.AppID)
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()
	log.Debugf("update role sql:%v", sql)

	_, err = db.Exec(sql)
	return errors.Trace(err)
}

func updateUser(u meta.User) error {
	sql := fmt.Sprintf("update user set name='%s', email='%s', mtime=now() where id=%d and app_id=%d", u.Name, u.Email, u.ID, u.AppID)
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()
	log.Debugf("update user sql:%v", sql)

	_, err = db.Exec(sql)
	return errors.Trace(err)
}

func exec(sql string) (int64, error) {
	db, err := mdb.GetConnection()
	if err != nil {
		return -1, errors.Trace(err)
	}
	defer db.Close()

	ret, err := db.Exec(sql)
	if err != nil {
		return -1, errors.Trace(err)
	}

	return ret.RowsAffected()
}

func validate(sql string) (bool, error) {
	db, err := mdb.GetConnection()
	if err != nil {
		return false, errors.Trace(err)
	}
	defer db.Close()

	rows, err := db.Query(sql)
	if err != nil {
		return false, errors.Trace(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return false, nil
	}

	return true, nil
}
