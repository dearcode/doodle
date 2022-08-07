package document

// Field 方法中的参数.
type Field struct {
	Name     string
	Type     string
	Required bool
	Comment  string
	Child    map[string]Field
}

// Method 接口中的一个方法.
type Method struct {
	Comment  string
	Request  map[string]Field
	Response map[string]Field
}

// Module 一个 module代表一个接口.
type Module struct {
	URL     string
	Methods map[string]Method
}
