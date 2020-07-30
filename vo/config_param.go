package vo

type Listener func(namespace, group, dataId, data string)

type ConfigParam struct {
	DataId   string `param:"dataId"`  //required
	Group    string `param:"group"`   //required
	Content  string `param:"content"` //required
	OnChange func(namespace, group, dataId, data string)
}

type SearchConfigParm struct {
	Search   string `param:"search"`
	DataId   string `param:"dataId"`
	Group    string `param:"group"`
	Tag      string `param:"tag"`
	AppName  string `param:"appName"`
	PageNo   int    `param:"pageNo"`
	PageSize int    `param:"pageSize"`
}
