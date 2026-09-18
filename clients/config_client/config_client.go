/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config_client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v3/common/security"

	"github.com/nacos-group/nacos-sdk-go/v3/clients/cache"
	"github.com/nacos-group/nacos-sdk-go/v3/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/v3/common/constant"
	nacos_inner_encryption "github.com/nacos-group/nacos-sdk-go/v3/common/encryption"
	"github.com/nacos-group/nacos-sdk-go/v3/common/filter"
	"github.com/nacos-group/nacos-sdk-go/v3/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v3/common/monitor"
	"github.com/nacos-group/nacos-sdk-go/v3/common/nacos_error"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_request"
	"github.com/nacos-group/nacos-sdk-go/v3/common/remote/rpc/rpc_response"
	"github.com/nacos-group/nacos-sdk-go/v3/inner/uuid"
	"github.com/nacos-group/nacos-sdk-go/v3/model"
	"github.com/nacos-group/nacos-sdk-go/v3/util"
	"github.com/nacos-group/nacos-sdk-go/v3/vo"
	"github.com/pkg/errors"
)

const (
	perTaskConfigSize = 3000
	executorErrDelay  = 5 * time.Second
)

// ErrConfigClientClosed is returned by ListenConfig once CloseClient has run:
// the listen executor goroutine is gone for good, so a newly (or
// concurrently) committed listener could never be served. Exported so
// callers can errors.Is against it rather than matching on error text.
var ErrConfigClientClosed = errors.New("config client is closed")

type ConfigClient struct {
	ctx    context.Context
	cancel context.CancelFunc
	nacos_client.INacosClient
	configFilterChainManager filter.IConfigFilterChain
	mutex                    sync.Mutex
	configProxy              IConfigProxy
	configCacheDir           string
	lastAllSyncTime          time.Time
	holder                   *configCacheHolder
	uid                      string
	listenExecute            chan struct{}
	isClosed                 bool
}

func NewConfigClientWithRamCredentialProvider(nc nacos_client.INacosClient, provider security.RamCredentialProvider) (*ConfigClient, error) {
	config := &ConfigClient{}
	config.ctx, config.cancel = context.WithCancel(context.Background())
	config.INacosClient = nc
	clientConfig, err := nc.GetClientConfig()
	if err != nil {
		config.cancel()
		return nil, err
	}
	serverConfig, err := nc.GetServerConfig()
	if err != nil {
		config.cancel()
		return nil, err
	}
	httpAgent, err := nc.GetHttpAgent()
	if err != nil {
		config.cancel()
		return nil, err
	}

	if err = initLogger(clientConfig); err != nil {
		config.cancel()
		return nil, err
	}
	clientConfig.CacheDir = clientConfig.CacheDir + string(os.PathSeparator) + "config"
	config.configCacheDir = clientConfig.CacheDir

	if config.configProxy, err = NewConfigProxyWithRamCredentialProvider(config.ctx, serverConfig, clientConfig, httpAgent, provider); err != nil {
		config.cancel()
		return nil, err
	}

	config.configFilterChainManager = filter.NewConfigFilterChainManager()

	if clientConfig.OpenKMS {
		kmsEncryptionHandler := nacos_inner_encryption.NewKmsHandler()
		nacos_inner_encryption.RegisterConfigEncryptionKmsPlugins(kmsEncryptionHandler, clientConfig)
		encryptionFilter := filter.NewDefaultConfigEncryptionFilter(kmsEncryptionHandler)
		err := filter.RegisterConfigFilterToChain(config.configFilterChainManager, encryptionFilter)
		if err != nil {
			logger.Error(err)
		}
	}

	uid, err := uuid.NewV4()
	if err != nil {
		config.cancel()
		return nil, err
	}

	config.uid = uid.String()
	config.holder = newConfigCacheHolder()
	// 1-buffered coalescing bell; see asyncNotifyListenConfig.
	config.listenExecute = make(chan struct{}, 1)
	config.startInternal()
	return config, err
}

func NewConfigClient(nc nacos_client.INacosClient) (*ConfigClient, error) {
	return NewConfigClientWithRamCredentialProvider(nc, nil)
}

func initLogger(clientConfig constant.ClientConfig) error {
	return logger.InitLogger(logger.BuildLoggerConfig(clientConfig))
}

func (client *ConfigClient) GetConfig(param vo.ConfigParam) (content string, err error) {
	content, encryptedDataKey, err := client.getConfigInner(param)
	if err != nil {
		return "", err
	}
	deepCopyParam := param.DeepCopy()
	deepCopyParam.EncryptedDataKey = encryptedDataKey
	deepCopyParam.Content = content
	deepCopyParam.UsageType = vo.ResponseType
	if err = client.configFilterChainManager.DoFilters(deepCopyParam); err != nil {
		return "", err
	}
	content = deepCopyParam.Content
	return content, nil
}

func (client *ConfigClient) getConfigInner(param vo.ConfigParam) (content, encryptedDataKey string, err error) {
	if len(param.DataId) <= 0 {
		err = errors.New("[client.GetConfig] param.dataId can not be empty")
		return "", "", err
	}
	if len(param.Group) <= 0 {
		param.Group = constant.DEFAULT_GROUP
	}

	clientConfig, _ := client.GetClientConfig()
	cacheKey := util.GetConfigCacheKey(param.DataId, param.Group, clientConfig.NamespaceId)
	content = cache.GetFailover(cacheKey, client.configCacheDir)
	if len(content) > 0 {
		logger.Warnf("%s %s %s is using failover content!", clientConfig.NamespaceId, param.Group, param.DataId)
		encryptedDataKey = cache.GetFailoverEncryptedDataKey(cacheKey, client.configCacheDir)
		return content, encryptedDataKey, nil
	}
	response, err := client.configProxy.queryConfig(param.DataId, param.Group, clientConfig.NamespaceId,
		clientConfig.TimeoutMs, false, client)
	if err != nil {
		logger.Errorf("get config from server error:%v, dataId=%s, group=%s, namespaceId=%s", err,
			param.DataId, param.Group, clientConfig.NamespaceId)

		if clientConfig.DisableUseSnapShot {
			return "", "", errors.Errorf("get config from remote nacos server fail, and is not allowed to read local file, err:%v", err)
		}

		cacheContent, cacheErr := cache.ReadConfigFromFile(cacheKey, client.configCacheDir)
		if cacheErr != nil {
			return "", "", errors.Errorf("read config from both server and cache fail, err=%v，dataId=%s, group=%s, namespaceId=%s",
				cacheErr, param.DataId, param.Group, clientConfig.NamespaceId)
		}

		if !strings.HasPrefix(param.DataId, nacos_inner_encryption.CipherPrefix) {
			return cacheContent, "", nil
		}
		encryptedDataKey, cacheErr = cache.ReadEncryptedDataKeyFromFile(cacheKey, client.configCacheDir)
		if cacheErr != nil {
			return "", "", errors.Errorf("read encryptedDataKey from server and cache fail, err=%v，dataId=%s, group=%s, namespaceId=%s",
				cacheErr, param.DataId, param.Group, clientConfig.NamespaceId)
		}

		logger.Warnf("read config from cache success, dataId=%s, group=%s, namespaceId=%s", param.DataId, param.Group, clientConfig.NamespaceId)
		return cacheContent, encryptedDataKey, nil
	}
	if response != nil && response.Response != nil && !response.IsSuccess() {
		return response.Content, response.EncryptedDataKey, errors.New(response.GetMessage())
	}
	encryptedDataKey = response.EncryptedDataKey
	content = response.Content
	return content, encryptedDataKey, nil
}

func (client *ConfigClient) PublishConfig(param vo.ConfigParam) (published bool, err error) {
	if len(param.DataId) <= 0 {
		err = errors.New("[client.PublishConfig] param.dataId can not be empty")
		return
	}
	if len(param.Content) <= 0 {
		err = errors.New("[client.PublishConfig] param.content can not be empty")
		return
	}

	if len(param.Group) <= 0 {
		param.Group = constant.DEFAULT_GROUP
	}

	param.UsageType = vo.RequestType
	if err = client.configFilterChainManager.DoFilters(&param); err != nil {
		return false, err
	}

	clientConfig, _ := client.GetClientConfig()
	request := rpc_request.NewConfigPublishRequest(param.Group, param.DataId, clientConfig.NamespaceId, param.Content, param.CasMd5)
	request.AdditionMap["tag"] = param.Tag
	request.AdditionMap["config_tags"] = param.ConfigTags
	request.AdditionMap["appName"] = param.AppName
	request.AdditionMap["betaIps"] = param.BetaIps
	request.AdditionMap["type"] = param.Type
	request.AdditionMap["src_user"] = param.SrcUser
	request.AdditionMap["encryptedDataKey"] = param.EncryptedDataKey
	rpcClient := client.configProxy.getRpcClient(client)
	response, err := client.configProxy.requestProxy(rpcClient, request, constant.DEFAULT_TIMEOUT_MILLS)
	if err != nil {
		return false, err
	}
	if response != nil {
		return client.buildResponse(response)
	}
	return false, err
}

func (client *ConfigClient) DeleteConfig(param vo.ConfigParam) (deleted bool, err error) {
	if len(param.DataId) <= 0 {
		err = errors.New("[client.DeleteConfig] param.dataId can not be empty")
	}
	if len(param.Group) <= 0 {
		param.Group = constant.DEFAULT_GROUP
	}
	if err != nil {
		return false, err
	}
	clientConfig, _ := client.GetClientConfig()
	request := rpc_request.NewConfigRemoveRequest(param.Group, param.DataId, clientConfig.NamespaceId)
	rpcClient := client.configProxy.getRpcClient(client)
	response, err := client.configProxy.requestProxy(rpcClient, request, constant.DEFAULT_TIMEOUT_MILLS)
	if err != nil {
		return false, err
	}
	if response != nil {
		return client.buildResponse(response)
	}
	return false, err
}

// CancelListenConfig cancels a previously registered listen for the given
// key. If the key was never listened on, this is a no-op returning nil. The
// entry (if any) is marked discarded rather than removed outright; reconciling
// removal from the holder is left to removeIfDiscarded/the executor.
func (client *ConfigClient) CancelListenConfig(param vo.ConfigParam) (err error) {
	clientConfig, err := client.GetClientConfig()
	if err != nil {
		logger.Errorf("[checkConfigInfo.GetClientConfig] failed,err:%+v", err)
		return
	}
	key := util.GetConfigCacheKey(param.DataId, param.Group, clientConfig.NamespaceId)
	if cData, ok := client.holder.get(key); ok {
		cData.markDiscard()
		client.asyncNotifyListenConfig()
	}
	logger.Infof("Cancel listen config DataId:%s Group:%s", param.DataId, param.Group)
	return nil
}

// ListenConfig registers OnChange to be notified about changes to the
// dataId/group/namespace identified by param. Calling it repeatedly for the
// same key appends additional independent listeners rather than replacing
// the previous one; calling it again after CancelListenConfig revives the
// entry.
func (client *ConfigClient) ListenConfig(param vo.ConfigParam) (err error) {
	if len(param.DataId) <= 0 {
		err = errors.New("[client.ListenConfig] DataId can not be empty")
		return err
	}
	if len(param.Group) <= 0 {
		err = errors.New("[client.ListenConfig] Group can not be empty")
		return err
	}
	client.mutex.Lock()
	closed := client.isClosed
	client.mutex.Unlock()
	if closed {
		return ErrConfigClientClosed
	}
	clientConfig, err := client.GetClientConfig()
	if err != nil {
		err = errors.New("[checkConfigInfo.GetClientConfig] failed")
		return err
	}

	key := util.GetConfigCacheKey(param.DataId, param.Group, clientConfig.NamespaceId)
	// Computed ahead of getOrCreate: getOrCreate already holds the holder's
	// write lock while invoking seed, so calling back into holder.count()
	// (which itself locks) from inside seed would deadlock.
	taskId := client.holder.count() / perTaskConfigSize

	cData := client.holder.getOrCreate(key, func() *cacheData {
		content, innerErr := cache.ReadConfigFromFile(key, client.configCacheDir)
		if innerErr != nil {
			logger.Warn(innerErr)
		}
		encryptedDataKey, _ := cache.ReadEncryptedDataKeyFromFile(key, client.configCacheDir)
		var md5Str string
		if len(content) > 0 {
			md5Str = util.Md5(content)
		}
		return &cacheData{
			isInitializing:   true,
			dataId:           param.DataId,
			group:            param.Group,
			tenant:           clientConfig.NamespaceId,
			content:          content,
			md5:              md5Str,
			encryptedDataKey: encryptedDataKey,
			taskId:           taskId,
		}
	})

	// Revive (discard=false) and append must happen in one critical section:
	// see reviveAndAddListener's doc comment for why a separate lock/unlock
	// around isInitializing/md5 followed by a separately-locked addListener
	// call is unsafe against a concurrent CancelListenConfig.
	cData.reviveAndAddListener(param.OnChange)

	// CloseClient may land concurrently, between the isClosed check above and
	// the commit just performed -- neither the check nor the seed's disk I/O
	// nor the commit itself holds client.mutex.
	return client.rejectIfClosedAfterCommit(cData)
}

// rejectIfClosedAfterCommit re-checks isClosed under client.mutex -- the same
// lock CloseClient sets isClosed under, and isClosed only ever transitions
// false->true -- immediately after a ListenConfig commit (getOrCreate +
// reviveAndAddListener). This linearizes the commit against CloseClient: if a
// close has already landed by this point, the commit is rolled back via
// cData.markDiscard() rather than left live on a client whose listen executor
// is gone for good.
func (client *ConfigClient) rejectIfClosedAfterCommit(cData *cacheData) error {
	client.mutex.Lock()
	closed := client.isClosed
	client.mutex.Unlock()
	if closed {
		cData.markDiscard()
		return ErrConfigClientClosed
	}
	return nil
}

func (client *ConfigClient) SearchConfig(param vo.SearchConfigParam) (*model.ConfigPage, error) {
	return client.searchConfigInner(param)
}

func (client *ConfigClient) CloseClient() {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if client.isClosed {
		return
	}
	client.configProxy.getRpcClient(client).Shutdown()
	client.cancel()
	client.isClosed = true
}

func (client *ConfigClient) searchConfigInner(param vo.SearchConfigParam) (*model.ConfigPage, error) {
	if param.Search != "accurate" && param.Search != "blur" {
		return nil, errors.New("[client.searchConfigInner] param.search must be accurate or blur")
	}
	if param.PageNo <= 0 {
		param.PageNo = 1
	}
	if param.PageSize <= 0 {
		param.PageSize = 10
	}
	clientConfig, _ := client.GetClientConfig()
	configItems, err := client.configProxy.searchConfigProxy(param, clientConfig.NamespaceId, clientConfig.AccessKey, clientConfig.SecretKey)
	if err != nil {
		logger.Errorf("search config from server error:%+v ", err)
		if _, ok := err.(*nacos_error.NacosError); ok {
			nacosErr := err.(*nacos_error.NacosError)
			if nacosErr.ErrorCode() == "404" {
				return nil, errors.New("config not found")
			}
			if nacosErr.ErrorCode() == "403" {
				return nil, errors.New("get config forbidden")
			}
		}
		return nil, err
	}
	return configItems, nil
}

func (client *ConfigClient) startInternal() {
	go func() {
		timer := time.NewTimer(executorErrDelay)
		defer timer.Stop()
		for {
			select {
			case <-client.listenExecute:
				client.executeConfigListen()
			case <-timer.C:
				client.executeConfigListen()
			case <-client.ctx.Done():
				return
			}
			timer.Reset(executorErrDelay)
		}
	}()
}

// executeConfigListen runs one round of the listen executor. Each round:
//  1. sends a Listen=false batch (grouped by taskId) for every discarded
//     entry, and reaps each key via holder.removeIfDiscarded once its batch
//     gets a successful response -- entries whose cancel batch errors or
//     comes back non-success are left in place for the next round to retry;
//  2. sends a Listen=true batch for every non-discarded entry that is either
//     not in sync with the server or due for a full resync;
//  3. for every key reported in ChangedConfigs, refreshes it via
//     refreshContentAndCheck;
//  4. marks every other entry in that listen batch as isSyncWithServer=true.
//
// Cancel batches are sent before listen batches each round so a key that is
// simultaneously being cancelled and re-listened (a revive racing with an
// in-flight cancel) is decided by removeIfDiscarded's own re-check rather
// than by request ordering.
func (client *ConfigClient) executeConfigListen() {
	var (
		needAllSync    = time.Since(client.lastAllSyncTime) >= constant.ALL_SYNC_INTERNAL
		hasChangedKeys = false
	)

	// Re-notify any listener whose watermark trails its entry's md5 with no
	// delivery in flight: deliveries are asynchronous, so a wrap can finish a
	// callback (or panic out of one) after its entry's content moved on, and
	// the change that moved it may not produce another server notification.
	// This round-entry sweep is that catch-up path (the delivery completion
	// also rings the bell, so the sweep usually runs promptly rather than on
	// the poll cadence). Entries with no lagging idle wrap are a cheap no-op.
	for _, cData := range client.holder.snapshot() {
		cData.notifyListeners(client.configFilterChainManager, client.asyncNotifyListenConfig)
	}

	listenBatch, cancelBatch := client.buildListenTask(needAllSync)

	for taskId, caches := range cancelBatch {
		request := buildConfigBatchListenRequest(caches, false)
		rpcClient := client.configProxy.createRpcClient(client.ctx, fmt.Sprintf("%d", taskId), client)
		iResponse, err := client.configProxy.requestProxy(rpcClient, request, 3000)
		if err != nil {
			logger.Warnf("ConfigBatchListenRequest(cancel) failure, err:%v", err)
			continue
		}
		if iResponse == nil {
			logger.Warnf("ConfigBatchListenRequest(cancel) failure, response is nil")
			continue
		}
		if !iResponse.IsSuccess() {
			logger.Warnf("ConfigBatchListenRequest(cancel) failure, error code:%d", iResponse.GetErrorCode())
			continue
		}
		for _, cData := range caches {
			cData.mu.Lock()
			key := util.GetConfigCacheKey(cData.dataId, cData.group, cData.tenant)
			cData.mu.Unlock()
			client.holder.removeIfDiscarded(key)
		}
	}

	for taskId, caches := range listenBatch {
		request := buildConfigBatchListenRequest(caches, true)
		rpcClient := client.configProxy.createRpcClient(client.ctx, fmt.Sprintf("%d", taskId), client)
		iResponse, err := client.configProxy.requestProxy(rpcClient, request, 3000)
		if err != nil {
			logger.Warnf("ConfigBatchListenRequest failure, err:%v", err)
			continue
		}
		if iResponse == nil {
			logger.Warnf("ConfigBatchListenRequest failure, response is nil")
			continue
		}
		if !iResponse.IsSuccess() {
			logger.Warnf("ConfigBatchListenRequest failure, error code:%d", iResponse.GetErrorCode())
			continue
		}
		response, ok := iResponse.(*rpc_response.ConfigChangeBatchListenResponse)
		if !ok {
			continue
		}

		if len(response.ChangedConfigs) > 0 {
			hasChangedKeys = true
		}
		changeKeys := make(map[string]struct{}, len(response.ChangedConfigs))
		for _, v := range response.ChangedConfigs {
			changeKey := util.GetConfigCacheKey(v.DataId, v.Group, v.Tenant)
			changeKeys[changeKey] = struct{}{}
			if cData, ok := client.holder.get(changeKey); ok {
				cData.mu.Lock()
				isInitializing := cData.isInitializing
				cData.mu.Unlock()
				client.refreshContentAndCheck(cData, !isInitializing)
			}
		}

		for _, cData := range caches {
			cData.mu.Lock()
			changeKey := util.GetConfigCacheKey(cData.dataId, cData.group, cData.tenant)
			if _, changed := changeKeys[changeKey]; !changed {
				cData.isSyncWithServer = true
			} else {
				cData.isInitializing = true
			}
			cData.mu.Unlock()
		}
	}

	if needAllSync {
		client.lastAllSyncTime = time.Now()
	}

	if hasChangedKeys {
		client.asyncNotifyListenConfig()
	}
	monitor.GetListenConfigCountMonitor().Set(float64(client.holder.count()))
}

func buildConfigBatchListenRequest(caches []*cacheData, listen bool) *rpc_request.ConfigBatchListenRequest {
	request := rpc_request.NewConfigBatchListenRequest(len(caches))
	request.Listen = listen
	for _, cData := range caches {
		cData.mu.Lock()
		ctx := model.ConfigListenContext{Group: cData.group, Md5: cData.md5, DataId: cData.dataId, Tenant: cData.tenant}
		cData.mu.Unlock()
		request.ConfigListenContexts = append(request.ConfigListenContexts, ctx)
	}
	return request
}

func (client *ConfigClient) refreshContentAndCheck(cData *cacheData, notify bool) {
	cData.mu.Lock()
	dataId, group, tenant := cData.dataId, cData.group, cData.tenant
	cData.mu.Unlock()

	configQueryResponse, err := client.configProxy.queryConfig(dataId, group, tenant,
		constant.DEFAULT_TIMEOUT_MILLS, notify, client)
	if err != nil {
		logger.Errorf("refresh content and check md5 fail ,dataId=%s,group=%s,tenant=%s ", dataId, group, tenant)
		return
	}
	if configQueryResponse != nil && configQueryResponse.Response != nil && !configQueryResponse.IsSuccess() {
		logger.Errorf("refresh cached config from server error:%v, dataId=%s, group=%s", configQueryResponse.GetMessage(),
			dataId, group)
		return
	}

	cData.mu.Lock()
	cData.content = configQueryResponse.Content
	cData.contentType = configQueryResponse.ContentType
	cData.encryptedDataKey = configQueryResponse.EncryptedDataKey
	if notify {
		logger.Infof("[config_rpc_client] [data-received] dataId=%s, group=%s, tenant=%s, md5=%s, content=%s, type=%s",
			dataId, group, tenant, cData.md5, util.TruncateContent(cData.content), cData.contentType)
	}
	cData.md5 = util.Md5(cData.content)
	cData.mu.Unlock()

	cData.notifyListeners(client.configFilterChainManager, client.asyncNotifyListenConfig)
}

// buildListenTask partitions the current holder snapshot into two batches,
// grouped by taskId: cancelBatch holds discarded entries (destined for a
// Listen=false request), and listenBatch holds every other entry that is
// either not in sync with the server or due for a full resync (destined for
// a Listen=true request). An entry that is both in sync and not due for
// resync needs no request this round and is omitted from both maps.
func (client *ConfigClient) buildListenTask(needAllSync bool) (listenBatch, cancelBatch map[int][]*cacheData) {
	listenBatch = make(map[int][]*cacheData, 8)
	cancelBatch = make(map[int][]*cacheData, 8)

	for _, cData := range client.holder.snapshot() {
		cData.mu.Lock()
		discard := cData.discard
		isSyncWithServer := cData.isSyncWithServer
		taskId := cData.taskId
		cData.mu.Unlock()

		if discard {
			cancelBatch[taskId] = append(cancelBatch[taskId], cData)
			continue
		}
		if !isSyncWithServer || needAllSync {
			listenBatch[taskId] = append(listenBatch[taskId], cData)
		}
	}
	return listenBatch, cancelBatch
}

// asyncNotifyListenConfig wakes the listen executor without blocking the
// caller. listenExecute is a 1-buffered coalescing bell (same shape as the
// naming FuzzyWatch holder's Bell): if a wake-up is already pending, this
// one merges into it -- the executor scans everything each round anyway, so
// N pending bells and one pending bell trigger identical work. The
// non-blocking send never spawns a goroutine and never leaks, no matter how
// many notifications land while the executor is busy or after the client is
// closed.
func (client *ConfigClient) asyncNotifyListenConfig() {
	select {
	case client.listenExecute <- struct{}{}:
	default:
	}
}

func (client *ConfigClient) buildResponse(response rpc_response.IResponse) (bool, error) {
	if response.IsSuccess() {
		return response.IsSuccess(), nil
	}
	return false, errors.New(response.GetMessage())
}
