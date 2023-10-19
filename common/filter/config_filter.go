package filter

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"sync"
)

var (
	initConfigFilterChainManagerOnce        = &sync.Once{}
	defaultConfigFilterChainManagerInstance IConfigFilterChain
)

type IConfigFilterChain interface {
	AddFilter(IConfigFilter) error
	GetFilters() []IConfigFilter
	DoFilters(*vo.ConfigParam) error
	DoFilterByName(*vo.ConfigParam, string) error
}

type IConfigFilter interface {
	DoFilter(*vo.ConfigParam) error
	GetOrder() int
	GetFilterName() string
}

func init() {
	err := RegisterConfigFilter(GetDefaultConfigFilterChainManager(), GetDefaultConfigEncryptionFilter())
	if err != nil {
		logger.Errorf("failed to register configFilter[%s] to DefaultConfigFilterChainManager",
			GetDefaultConfigEncryptionFilter().GetFilterName())
		return
	} else {
		logger.Infof("successfully register ConfigFilter[%s] to DefaultConfigFilterChainManager", GetDefaultConfigEncryptionFilter().GetFilterName())
	}
}

func GetDefaultConfigFilterChainManager() IConfigFilterChain {
	if defaultConfigFilterChainManagerInstance == nil {
		initConfigFilterChainManagerOnce.Do(func() {
			defaultConfigFilterChainManagerInstance = newDefaultConfigFilterChainManager()
			logger.Info("successfully create DefaultConfigFilterChainManager")
		})
	}
	return defaultConfigFilterChainManagerInstance
}

func newDefaultConfigFilterChainManager() *DefaultConfigFilterChainManager {
	return &DefaultConfigFilterChainManager{
		configFilterPriorityQueue: make([]IConfigFilter, 0, 2),
	}
}

type DefaultConfigFilterChainManager struct {
	configFilterPriorityQueue
}

func (m *DefaultConfigFilterChainManager) AddFilter(filter IConfigFilter) error {
	return m.configFilterPriorityQueue.addFilter(filter)
}

func (m *DefaultConfigFilterChainManager) GetFilters() []IConfigFilter {
	return m.configFilterPriorityQueue
}

func (m *DefaultConfigFilterChainManager) DoFilters(param *vo.ConfigParam) error {
	for index := 0; index < len(m.GetFilters()); index++ {
		if err := m.GetFilters()[index].DoFilter(param); err != nil {
			return err
		}
	}
	return nil
}

func (m *DefaultConfigFilterChainManager) DoFilterByName(param *vo.ConfigParam, name string) error {
	for index := 0; index < len(m.GetFilters()); index++ {
		if m.GetFilters()[index].GetFilterName() == name {
			if err := m.GetFilters()[index].DoFilter(param); err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("cannot find the filter[%s]", name)
}

func RegisterConfigFilter(chain IConfigFilterChain, filter IConfigFilter) error {
	return chain.AddFilter(filter)
}

type configFilterPriorityQueue []IConfigFilter

func (c *configFilterPriorityQueue) addFilter(filter IConfigFilter) error {
	var pos int = len(*c)
	for i := 0; i < len(*c); i++ {
		if filter.GetFilterName() == (*c)[i].GetFilterName() {
			return nil
		}
		if filter.GetOrder() < (*c)[i].GetOrder() {
			pos = i
			break
		}
	}
	if pos == len(*c) {
		*c = append((*c)[:], filter)
	} else {
		temp := append((*c)[:pos], filter)
		*c = append(temp[:], (*c)[pos:]...)
	}
	return nil
}
