package vo

type Listener func(namespace, group, dataId, data string)

type ConfigParam struct {
	DataId   string `param:"dataId"`
	Group    string `param:"group"`
	Content  string `param:"content"`
	Tag      string `param:"tag"`
	AppName  string `param:"appName"`
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
