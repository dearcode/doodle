package service

import (
	"html/template"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"

	"dearcode.net/doodle/pkg/service/debug"
)

const (
	docTemplate = `<!DOCTYPE html>
    <html>
    <head>
    <style type="text/css">
    table { color:#333333; border-width: 1px; border-color: #666666; border-collapse: collapse; }
    table th { border-width: 1px; padding: 5px; border-style: solid; border-color: #666666; background-color: #dedede; }
    table td { border-width: 1px; padding: 5px; border-style: solid; border-color: #666666; background-color: #ffffff; }
    </style>
    </head>
    <body>

    {{ range .Methods }}

    <p> <b>URL:</b> {{ .URL }} </p>
    <p> <b>说明:</b> {{ .Comment }} </p>
    <p> <b>方法:</b> {{ .Method }} </p>

    <p> <b>请求参数:</b>
    <table>
    <tr><th>名称</th><th>类型</th><th>必选</th><th>说明</th></tr>
    {{ range .Request }}
    <tr><td>{{.Name}}</td><td>{{.Type}}</td><td>{{.Required}}</td><td>{{.Comment}}</td></tr>
    {{ end }}
    </table>
    </p>

    <p> <b>返回参数:</b>
    <table>
    <tr><th>名称</th><th>类型</th><th>必选</th><th>说明</th></tr>
    {{ range .Response }}
    <tr><td>{{.Name}}</td><td>{{.Type}}</td><td>{{.Required}}</td><td>{{.Comment}}</td></tr>
    {{ end }}
    </table>
    </p>


    {{ end }}

    </body>
    </html>`
)

var (
	docExport = make(map[string]string)
)

type docView struct {
	Methods []docViewMethod
}

type docViewMethod struct {
	Name     string
	URL      string
	Method   string
	Comment  string
	Request  []docViewField
	Response []docViewField
}

type docViewField struct {
	Name     template.HTML
	Type     string
	Required bool
	Comment  string
}

func (d *document) view() docView {
	d.mu.Lock()
	defer d.mu.Unlock()

	dv := docView{}

	for mk, mv := range d.Modules {
		for mmk, mmv := range mv.Methods {
			dvm := docViewMethod{
				Name:    mk,
				Method:  mmk,
				URL:     mv.URL,
				Comment: mmv.Comment,
			}

			for _, rf := range mmv.Request {
				f := docViewField{
					Name:     template.HTML(rf.Name),
					Type:     rf.Type,
					Required: rf.Required,
					Comment:  rf.Comment,
				}
				dvm.Request = append(dvm.Request, f)

				for _, c := range rf.Child {
					dvm.Request = append(dvm.Request, c.views(1)...)
				}
			}

			for _, rf := range mmv.Response {
				f := docViewField{
					Name:     template.HTML(rf.Name),
					Type:     rf.Type,
					Required: rf.Required,
					Comment:  rf.Comment,
				}
				dvm.Response = append(dvm.Response, f)

				for _, c := range rf.Child {
					dvm.Response = append(dvm.Response, c.views(1)...)
				}
			}

			dv.Methods = append(dv.Methods, dvm)
		}
	}

	return dv
}

func (f *field) views(level int) []docViewField {
	dvfs := []docViewField{{
		Name:     template.HTML(strings.Repeat("&nbsp;&nbsp;&sdot;&nbsp;&nbsp;", level) + f.Name),
		Type:     f.Type,
		Required: f.Required,
		Comment:  f.Comment,
	}}

	for _, v := range f.Child {
		dvfs = append(dvfs, v.views(level+1)...)
	}

	return dvfs
}

func (d *docView) GET(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("t").Parse(docTemplate)
	if err != nil {
		log.Errorf("ParseGlob error:%v", err)
		server.Abort(w, err.Error())
		return
	}

	if err := t.Execute(w, *d); err != nil {
		log.Errorf("Execute error:%v", err)
		server.Abort(w, err.Error())
	}
}

type document struct {
	Modules map[string]module
	mu      sync.Mutex
}

type module struct {
	URL     string
	Methods map[string]*method
}

type field struct {
	Name      string
	Type      string
	Required  bool
	Child     map[string]*field `json:",omitempty"`
	Comment   string
	anonymous bool
}

type method struct {
	Comment  string
	Request  map[string]*field
	Response map[string]*field
}

func newDocument() document {
	return document{Modules: make(map[string]module)}
}

func (d *document) GET(w http.ResponseWriter, r *http.Request) {
	server.SendData(w, d.Modules)
}

func (d *document) add(name, url string, rm reflect.Method) {
	d.mu.Lock()
	defer d.mu.Unlock()

	log.Debugf("Module:%v Method:%v %v", name, rm.Name, rm.Type)
	md, ok := d.Modules[name]
	if !ok {
		md = module{Methods: make(map[string]*method), URL: url}
		d.Modules[name] = md
	}

	m, ok := md.Methods[rm.Name]
	if !ok {
		log.Debugf("name:%v, url:%v, method:%v", name, url, rm.Name)
		m = &method{
			Comment:  getExportComment(name, url, rm.Name),
			Request:  make(map[string]*field),
			Response: make(map[string]*field),
		}
		md.Methods[rm.Name] = m
	}

	m.parse(rm.Type.In(1), m.Request)
	m.parse(rm.Type.In(2), m.Response)

	m.merge(m.Request)
	m.merge(m.Response)
}

func (m *method) parse(arg reflect.Type, fm map[string]*field) {
	if arg.Kind() == reflect.Ptr {
		arg = arg.Elem()
	}
	for i := 0; i < arg.NumField(); i++ {
		sf := arg.Field(i)
		if sf.Type.String() == "service.RequestHeader" {
			continue
		}
		//log.Debugf("arg:%v, field:%v, type:%v", arg, sf.Name, sf.Type.String())
		fm[sf.Name] = newField(sf)
	}
}

// merge 合并匿名变量.
func (m *method) merge(fm map[string]*field) {
	ok := true

	for ok {
		ok = false
		//清理当前层匿名类.
		for k, v := range fm {
			if v.anonymous {
				//log.Debugf("remove k:%v", k)
				delete(fm, k)
				for fk, fv := range v.Child {
					fm[fk] = fv
				}
				ok = true
			}
		}

		//遍历所有子类.
		for _, v := range fm {
			if v.Child != nil {
				m.merge(v.Child)
			}
		}
	}
}

func newField(sf reflect.StructField) *field {
	f := &field{
		Comment:   sf.Tag.Get("comment"),
		Name:      sf.Name,
		Type:      sf.Type.String(),
		anonymous: sf.Anonymous,
	}

	if r := sf.Tag.Get("required"); r == "true" {
		f.Required = true
	}

	if n := strings.Split(sf.Tag.Get("json"), ",")[0]; n != "" {
		f.Name = n
	}

	st := sf.Type

	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}

	if st.Kind() == reflect.Slice {
		st = st.Elem()
	}

	if st.Kind() != reflect.Struct {
		return f
	}

	//log.Debugf("StructField:%v, st:%v", sf, st)

	for i := 0; i < st.NumField(); i++ {
		sf := st.Field(i)
		nf := newField(sf)
		if f.Child == nil {
			f.Child = make(map[string]*field)
		}
		f.Child[sf.Name] = nf
	}
	return f
}

// getExportComment 根据go doc生成的函数注释查询.
func getExportComment(name, url, method string) string {
	key := url[:len(url)-len(name)-2]
	key = debug.Project + key + "." + name + "." + method
	log.Debugf("project:%v, name:%v, url:%v, method:%v, key:%v", debug.Project, name, url, method, key)
	return docExport[key]
}
