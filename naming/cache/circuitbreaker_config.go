/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package cache

import (
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"
	"sync"
	"time"
)

const (
	CircuitBreakerName = "circuitBreakerConfig"
)

// circuitBreaker配置的cache接口
type CircuitBreakerCache interface {
	Cache

	// 根据ServiceID获取熔断配置
	GetCircuitBreakerConfig(id string) *model.ServiceWithCircuitBreaker
}

/**
 * @brief circuitBreaker的实现
 */
type circuitBreakerCache struct {
	storage     store.Store
	ids         *sync.Map
	lastTime    time.Time
	firstUpdate bool
}

/**
 * @brief 返回一个操作CircuitBreakerCache的对象
 */
func newCircuitBreakerCache(s store.Store) *circuitBreakerCache {
	return &circuitBreakerCache{
		storage: s,
	}
}

/**
 * @brief 实现Cache接口的函数
 */
func (cbc *circuitBreakerCache) initialize(opt map[string]interface{}) error {
	cbc.ids = new(sync.Map)
	cbc.lastTime = time.Unix(0, 0)
	cbc.firstUpdate = true
	if opt == nil {
		return nil
	}
	return nil
}

/**
 * @brief 实现Cache接口的函数
 */
func (cbc *circuitBreakerCache) update() error {
	out, err := cbc.storage.GetCircuitBreakerForCache(cbc.lastTime.Add(DefaultTimeDiff), cbc.firstUpdate)
	if err != nil {
		log.Errorf("[Cache] circuit breaker config cache update err:%s", err.Error())
		return err
	}

	cbc.firstUpdate = false
	return cbc.setCircuitBreaker(out)
}

/**
 * @brief 实现Cache接口的函数
 */
func (cbc *circuitBreakerCache) clear() error {
	cbc.ids = new(sync.Map)
	cbc.lastTime = time.Unix(0, 0)
	return nil
}

/**
 * @brief 实现资源名称
 */
func (cbc *circuitBreakerCache) name() string {
	return CircuitBreakerName
}

/**
 * @brief 根据serviceID获取熔断规则
 */
func (cbc *circuitBreakerCache) GetCircuitBreakerConfig(id string) *model.ServiceWithCircuitBreaker {
	if id == "" {
		return nil
	}

	value, ok := cbc.ids.Load(id)
	if !ok {
		return nil
	}

	return value.(*model.ServiceWithCircuitBreaker)
}

/**
 * @brief 更新store的数据到cache中
 */
func (cbc *circuitBreakerCache) setCircuitBreaker(cb []*model.ServiceWithCircuitBreaker) error {
	if len(cb) == 0 {
		return nil
	}

	lastTime := cbc.lastTime.Unix()
	for _, entry := range cb {
		if entry.ServiceID == "" {
			continue
		}

		if entry.ModifyTime.Unix() > lastTime {
			lastTime = entry.ModifyTime.Unix()
		}

		if entry.Valid == false {
			cbc.ids.Delete(entry.ServiceID)
			continue
		}

		cbc.ids.Store(entry.ServiceID, entry)
	}

	if cbc.lastTime.Unix() < lastTime {
		cbc.lastTime = time.Unix(lastTime, 0)
	}
	return nil
}

/**
 * @brief 获取熔断规则总数
 */
func (cbc *circuitBreakerCache) GetCircuitBreakerCount(f func(k, v interface{}) bool) {
	cbc.ids.Range(f)
}

/**
 * @brief 自注册到缓存列表
 */
func init() {
	RegisterCache(CircuitBreakerName, CacheCircuitBreaker)
}
