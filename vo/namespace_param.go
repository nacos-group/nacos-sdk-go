package vo

type CreateNamespaceParam struct {
	CustomNamespaceId string `param:"customNamespaceId"`
	NamespaceName     string `param:"namespaceName"` //required
	NamespaceDesc     string `param:"namespaceDesc"`
}

type ModifyNamespaceParam struct {
	NamespaceId   string `param:"namespaceId"`
	NamespaceName string `param:"namespaceName"` //required
	NamespaceDesc string `param:"namespaceDesc"`
}

type DeleteNamespaceParam struct {
	NamespaceId string `param:"namespaceId"`
}
