package distributor

type node struct {
	ID     int64
	Server string
	Ctime  string `db_default:"now()"`
}

type cluster struct {
	ID      int64
	Name    string
	Comment string
	Node    []node `db_table:"one2more"`
	Ctime   string `db_default:"now()"`
	Mtime   string `db_default:"now()"`
}
