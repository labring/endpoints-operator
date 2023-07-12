// Copyright © 2023 sealos.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"flag"
	"golang.org/x/time/rate"
	"time"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/ratelimiter"
)

const (
	defaultMinRetryDelay = 5 * time.Millisecond
	defaultMaxRetryDelay = 1000 * time.Second
	defaultQPS           = float64(10.0)
	defaultBurst         = 100
	flagMinRetryDelay    = "min-retry-delay"
	flagMaxRetryDelay    = "max-retry-delay"
	flagQPS              = "default-qps"
	flagBurst            = "default-burst"
)

// RateLimiterOptions used on reconcilers.
type RateLimiterOptions struct {
	MinRetryDelay time.Duration
	QPS           float64
	Burst         int
	MaxRetryDelay time.Duration
}

func (o *RateLimiterOptions) BindFlags(fs *flag.FlagSet) {
	fs.DurationVar(&o.MinRetryDelay, flagMinRetryDelay, defaultMinRetryDelay,
		"Specifies the minimum delay time before retrying the reconciliation of an object. This delay provides a buffer to prevent rapid-fire retries.")
	fs.DurationVar(&o.MaxRetryDelay, flagMaxRetryDelay, defaultMaxRetryDelay,
		"Specifies the maximum delay time before retrying the reconciliation of an object. This cap ensures that retry delays don't grow excessively long.")
	fs.Float64Var(&o.QPS, flagQPS, defaultQPS, "Sets the maximum allowed quantity of process units (batches) that can be processed per second. This limit helps maintain a controlled processing rate.")
	fs.IntVar(&o.Burst, flagBurst, defaultBurst, "Sets the maximum quantity of process units (batches) that can be processed in a burst. This limit helps control the processing rate during short periods of high activity.")
}

func GetRateLimiter(opts RateLimiterOptions) ratelimiter.RateLimiter {
	return workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(opts.MinRetryDelay, opts.MaxRetryDelay),
		// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(opts.QPS), opts.Burst)},
	)
}
