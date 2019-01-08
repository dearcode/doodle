package debug

import (
	"net/http"
	"net/http/pprof"
)

//Debug 用来输出pprof信息.
type Debug struct {
}

//GET pprof接口.
func (d *Debug) GET(w http.ResponseWriter, r *http.Request) {
	pprof.Index(w, r)
}
