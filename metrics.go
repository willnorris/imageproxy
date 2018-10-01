package imageproxy

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestServedFromCacheCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "requests_served_from_cache",
			Help: "Number of requests served from cache.",
		})
	imageTransformationSummary = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "image_transformation_seconds",
		Help: "Time taken for image transformations in seconds.",
	})
	compressionSummary = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "image_compression_seconds",
		Help: "Time taken for image compression in seconds.",
	})
	remoteImageFetchErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "remote_image_fetch_errors",
		Help: "Total image fetch failures",
	})
	httpRequestsResponseTime = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "http",
		Name:      "response_time_seconds",
		Help:      "Request response times",
	})
)

func init() {
	prometheus.MustRegister(compressionSummary)
	prometheus.MustRegister(imageTransformationSummary)
	prometheus.MustRegister(requestServedFromCacheCount)
	prometheus.MustRegister(remoteImageFetchErrors)
	prometheus.MustRegister(httpRequestsResponseTime)
}
