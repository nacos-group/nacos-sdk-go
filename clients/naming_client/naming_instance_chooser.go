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

package naming_client

import (
	"math/rand"
	"sort"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

type Chooser struct {
	data   []model.Instance
	totals []float64
	max    float64
}

type instance []model.Instance

func (a instance) Len() int {
	return len(a)
}

func (a instance) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a instance) Less(i, j int) bool {
	return a[i].Weight < a[j].Weight
}

// NewChooser initializes a new Chooser for picking from the provided Choices.
// Weights are accumulated as float64 so that fractional weights (e.g. 0.5),
// which are valid in Nacos, keep their proportion instead of being truncated.
func newChooser(instances []model.Instance) Chooser {
	sort.Sort(instance(instances))
	totals := make([]float64, len(instances))
	runningTotal := 0.0
	for i, c := range instances {
		runningTotal += c.Weight
		totals[i] = runningTotal
	}
	return Chooser{data: instances, totals: totals, max: runningTotal}
}

func (chs Chooser) pick() model.Instance {
	r := rand.Float64() * chs.max
	i := sort.Search(len(chs.totals), func(i int) bool { return chs.totals[i] > r })
	if i == len(chs.totals) {
		// only reachable when every weight is zero
		i = len(chs.totals) - 1
	}
	return chs.data[i]
}
