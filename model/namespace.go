package model

type Namespace struct {
	Namespace         string `json:"namespace"`
	NamespaceShowName string `json:"namespaceShowName"`
	Quota             int64  `json:"quota"`
	ConfigCount       int64  `json:"configCount"`
	NType             int    `json:"type"`
}

type NamespaceList struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    []*Namespace `json:"data"`
}

/*
customNamespaceId	字符串	是	命名空间ID
namespaceName	字符串	是	命名空间名
namespaceDesc	字符串	否	命名空间描述
*/
type NamespaceReq struct {
	CustomNamespaceId string `json:"customNamespaceId"`
	NamespaceName     string `json:"namespaceName"`
	NamespaceDesc     string `json:"namespaceDesc"`
}
