package model

type ConfigItem struct {
	Id      string `param:"id"`
	DataId  string `param:"dataId"`
	Group   string `param:"group"`
	Content string `param:"content"`
	Md5     string `param:"md5"`
	Tenant  string `param:"tenant"`
	Appname string `param:"appname"`
}
type ConfigPage struct {
	TotalCount     int          `param:"totalCount"`
	PageNumber     int          `param:"pageNumber"`
	PagesAvailable int          `param:"pagesAvailable"`
	PageItems      []ConfigItem `param:"pageItems"`
}
