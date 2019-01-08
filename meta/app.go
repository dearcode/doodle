package meta

import (
	"encoding/json"
	"strconv"
)

//MicroAPP 一个函数式应用.
type MicroAPP struct {
	ServiceKey string
	Host       string
	Port       int
	PID        int
	GitHash    string
	GitTime    string
	GitMessage string
}

//NewMicroAPP 一个应用.
func NewMicroAPP(host string, port int, key string, pid int, hash, time, message string) *MicroAPP {
	return &MicroAPP{
		ServiceKey: key,
		PID:        pid,
		Host:       host,
		Port:       port,
		GitHash:    hash,
		GitTime:    time,
		GitMessage: message,
	}
}

//Version 转换字符串类型的版本号为数值型.
func (m *MicroAPP) Version() int64 {
	v, _ := strconv.ParseInt(m.GitTime, 10, 64)
	return v
}

func (m *MicroAPP) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}
