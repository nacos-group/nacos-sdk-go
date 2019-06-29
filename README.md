## nacos-go
go语言版本的nacos client，支持服务发现和配置管理

### 客户端配置

* ClientConfig 客户端配置参数 
 
```go
constant.ClientConfig{
    TimeoutMs:      30 * 1000, //http请求超时时间，单位毫秒
    ListenInterval: 10 * 1000, //监听间隔时间，单位毫秒（仅在ConfigClient中有效）
    BeatInterval:   5 * 1000, //心跳间隔时间，单位毫秒（仅在ServiceClient中有效）
    NamespaceId:       "public", //nacos命名空间
    Endpoint:          "" //获取nacos节点ip的服务地址
    CacheDir:         "/data/nacos/cache", //缓存目录
    LogDIr:         "/data/nacos/log", //日志目录
    UpdateThreadNum:   20, //更新服务的线程数
    NotLoadCacheAtStart: true, //在启动时不读取本地缓存数据，true--不读取，false--读取
    UpdateCacheWhenEmpty: true, //当服务列表为空时是否更新本地缓存，true--更新,false--不更新
}
```


* ServerConfig nacos服务信息配置参数

```go
    constant.ServerConfig{{
		IpAddr:      "console.nacos.io", //nacos服务的ip地址 
		ContextPath: "/nacos", //nacos服务的上下文路径，默认是“/nacos” 
		Port:        80, //nacos服务端口
	}
```

<b>注：ServerConfig支持配置多个，在请求出错时，自动切换</b>

### 构造客户端

```go
// 可以没有，采用默认值
clientConfig := constant.ClientConfig{
    TimeoutMs:      30 * 1000,
    ListenInterval: 10 * 1000,
    BeatInterval:   5 * 1000,
    LogDir: "/nacos/logs",
    CacheDir: "/nacos/cache",
} 

// 至少一个
serverConfigs := []constant.ServerConfig{
    {
        IpAddr:      "console1.nacos.io",
        ContextPath: "/nacos",
        Port:        80,
    },
    {
    	IpAddr:      "console2.nacos.io",
    	ContextPath: "/nacos",
    	Port:        80,
    },
}

namingClient, err := clients.CreateNamingClient(map[string]interface{}{
	"serverConfigs": serverConfigs,
	"clientConfig":  clientConfig,
})

configClient, err := clients.CreateConfigClient(map[string]interface{}{
	"serverConfigs": serverConfigs,
	"clientConfig":  clientConfig,
})
    
```


### 服务发现
    
* 注册服务实例：RegisterInstance

```go

success, _ := namingClient.RegisterInstance(vo.RegisterInstanceParam{
    Ip:          "10.0.0.11",
    Port:        8848,
    ServiceName: "demo.go",
    Weight:      10,
    ClusterName: "a",
    Enable:      true,
    Healthy:     true,
    Ephemeral:   true,
})

```
  
* 注销服务实例：DeregisterInstance

```go

success, _ := namingClient.DeregisterInstance(vo.RegisterInstanceParam{
    Ip:          "10.0.0.11",
    Port:        8848,
    ServiceName: "demo.go",
    ClusterName: "a",
    Ephemeral:   true,
})

```
  
* 获取服务：GetService

```go

service, _ := namingClient.GetService(vo.GetServiceParam{
    ServiceName: "demo.go",
    Clusters:    []string{"a"},
})

```

* 获取所有的实例列表：SelectAllInstances

```go

instances, err := namingClient.SelectAllInstances(vo.SelectAllInstancesParam{
    ServiceName: "demo.go",
    Clusters:    []string{"a"},
})

```
 
* 获取实例列表：SelectInstances

```go

instances, err := namingClient.SelectInstances(vo.SelectInstancesParam{
    ServiceName: "demo.go",
    Clusters:    []string{"a"},
    HealthyOnly: true,
})

```

* 获取一个健康的实例（加权轮训负载均衡）：SelectOneHealthyInstance

```go

instance, err := namingClient.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{
    ServiceName: "demo.go",
    Clusters:    []string{"a"},
})

```

* 服务监听：Subscribe

```go

namingClient.Subscribe(vo.SubscribeParam{
    ServiceName: "demo.go",
    Clusters:    []string{"a"},
    SubscribeCallback: func(services []model.SubscribeService, err error) {
        log.Printf("\n\n callback return services:%s \n\n", utils.ToJsonString(services))
    },
})

```

* 取消服务监听：Unsubscribe

```go

namingClient.Unsubscribe(vo.SubscribeParam{
    ServiceName: "demo.go",
    Clusters:    []string{"a"},
    SubscribeCallback: func(services []model.SubscribeService, err error) {
        log.Printf("\n\n callback return services:%s \n\n", utils.ToJsonString(services))
    },
})

```

### 配置管理

* 发布配置：PublishConfig

```go

success, err := configClient.PublishConfig(vo.ConfigParam{
    DataId:  "dataId",
    Group:   "group",
    Content: "hello world!222222"})

```

* 删除配置：DeleteConfig

```go

success, err = configClient.DeleteConfig(vo.ConfigParam{
    DataId: "dataId",
    Group:  "group"})

```

* 获取配置：GetConfig

```go

content, err := configClient.GetConfig(vo.ConfigParam{
    DataId: "dataId",
    Group:  "group"})

```

* 监听配置：ListenConfig

```go

configClient.ListenConfig(vo.ConfigParam{
    DataId: "dataId",
    Group:  "group",
    OnChange: func(namespace, group, dataId, data string) {
        fmt.Println("group:" + group + ", dataId:" + dataId + ", data:" + data)
	},
})

```