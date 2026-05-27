// Package registry - imports.go registers all pattern detectors.
// Each pattern package's NewDetector() is called and registered here,
// ensuring the registry is populated when consumers import this package.
package registry

import (
	batch_processing "github.com/alpardfm/cost-efficient-go/patterns/batch-processing"
	caching_strategies "github.com/alpardfm/cost-efficient-go/patterns/caching-strategies"
	channel_patterns "github.com/alpardfm/cost-efficient-go/patterns/channel-patterns"
	connection_pooling "github.com/alpardfm/cost-efficient-go/patterns/connection-pooling"
	context_cancellation "github.com/alpardfm/cost-efficient-go/patterns/context-cancellation"
	efficient_logging "github.com/alpardfm/cost-efficient-go/patterns/efficient-logging"
	error_handling "github.com/alpardfm/cost-efficient-go/patterns/error-handling"
	goroutine_leak "github.com/alpardfm/cost-efficient-go/patterns/goroutine-leak"
	http_client_optimization "github.com/alpardfm/cost-efficient-go/patterns/http-client-optimization"
	interface_dispatch "github.com/alpardfm/cost-efficient-go/patterns/interface-dispatch"
	json_processing "github.com/alpardfm/cost-efficient-go/patterns/json-processing"
	map_internals "github.com/alpardfm/cost-efficient-go/patterns/map-internals"
	profiling_benchmarking "github.com/alpardfm/cost-efficient-go/patterns/profiling-benchmarking"
	query_optimization "github.com/alpardfm/cost-efficient-go/patterns/query-optimization"
	redis_pipeline "github.com/alpardfm/cost-efficient-go/patterns/redis-pipeline"
	slice_performance "github.com/alpardfm/cost-efficient-go/patterns/slice-performance"
	string_building "github.com/alpardfm/cost-efficient-go/patterns/string-building"
	struct_alignment "github.com/alpardfm/cost-efficient-go/patterns/struct-alignment"
	sync_pool "github.com/alpardfm/cost-efficient-go/patterns/sync-pool"
	worker_pool "github.com/alpardfm/cost-efficient-go/patterns/worker-pool"
)

func init() {
	Register(batch_processing.NewDetector())
	Register(caching_strategies.NewDetector())
	Register(channel_patterns.NewDetector())
	Register(connection_pooling.NewDetector())
	Register(context_cancellation.NewDetector())
	Register(efficient_logging.NewDetector())
	Register(error_handling.NewDetector())
	Register(goroutine_leak.NewDetector())
	Register(http_client_optimization.NewDetector())
	Register(interface_dispatch.NewDetector())
	Register(json_processing.NewDetector())
	Register(map_internals.NewDetector())
	Register(profiling_benchmarking.NewDetector())
	Register(query_optimization.NewDetector())
	Register(redis_pipeline.NewDetector())
	Register(slice_performance.NewDetector())
	Register(string_building.NewDetector())
	Register(struct_alignment.NewDetector())
	Register(sync_pool.NewDetector())
	Register(worker_pool.NewDetector())
}
