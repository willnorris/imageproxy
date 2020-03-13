// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package tracer

import (
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/globalconfig"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/version"
)

// config holds the tracer configuration.
type config struct {
	// debug, when true, writes details to logs.
	debug bool

	// serviceName specifies the name of this application.
	serviceName string

	// sampler specifies the sampler that will be used for sampling traces.
	sampler Sampler

	// agentAddr specifies the hostname and port of the agent where the traces
	// are sent to.
	agentAddr string

	// globalTags holds a set of tags that will be automatically applied to
	// all spans.
	globalTags map[string]interface{}

	// transport specifies the Transport interface which will be used to send data to the agent.
	transport transport

	// propagator propagates span context cross-process
	propagator Propagator

	// httpRoundTripper defines the http.RoundTripper used by the agent transport.
	httpRoundTripper http.RoundTripper

	// hostname is automatically assigned when the DD_TRACE_REPORT_HOSTNAME is set to true,
	// and is added as a special tag to the root span of traces.
	hostname string

	// logger specifies the logger to use when printing errors. If not specified, the "log" package
	// will be used.
	logger ddtrace.Logger

	// runtimeMetrics specifies whether collection of runtime metrics is enabled.
	runtimeMetrics bool

	// dogstatsdAddr specifies the address to connect for sending metrics to the
	// Datadog Agent. If not set, it defaults to "localhost:8125" or to the
	// combination of the environment variables DD_AGENT_HOST and DD_DOGSTATSD_PORT.
	dogstatsdAddr string

	// statsd is used for tracking metrics associated with the runtime and the tracer.
	statsd statsdClient

	// samplingRules contains user-defined rules determine the sampling rate to apply
	// to spans.
	samplingRules []SamplingRule

	// tickChan specifies a channel which will receive the time every time the tracer must flush.
	// It defaults to time.Ticker; replaced in tests.
	tickChan <-chan time.Time
}

// StartOption represents a function that can be provided as a parameter to Start.
type StartOption func(*config)

// defaults sets the default values for a config.
func defaults(c *config) {
	c.serviceName = filepath.Base(os.Args[0])
	c.sampler = NewAllSampler()
	c.agentAddr = defaultAddress

	statsdHost, statsdPort := "localhost", "8125"
	if v := os.Getenv("DD_AGENT_HOST"); v != "" {
		statsdHost = v
	}
	if v := os.Getenv("DD_DOGSTATSD_PORT"); v != "" {
		statsdPort = v
	}
	c.dogstatsdAddr = net.JoinHostPort(statsdHost, statsdPort)

	if os.Getenv("DD_TRACE_REPORT_HOSTNAME") == "true" {
		var err error
		c.hostname, err = os.Hostname()
		if err != nil {
			log.Warn("unable to look up hostname: %v", err)
		}
	}
	if v := os.Getenv("DD_ENV"); v != "" {
		WithEnv(v)(c)
	}
}

func statsTags(c *config) []string {
	tags := []string{
		"lang:go",
		"version:" + version.Tag,
		"lang_version:" + runtime.Version(),
	}
	if c.serviceName != "" {
		tags = append(tags, "service:"+c.serviceName)
	}
	if c.hostname != "" {
		tags = append(tags, "host:"+c.hostname)
	}
	if v, ok := c.globalTags[ext.Environment]; ok {
		if vv, ok := v.(string); ok {
			tags = append(tags, "env:"+vv)
		}
	}
	return tags
}

// WithLogger sets logger as the tracer's error printer.
func WithLogger(logger ddtrace.Logger) StartOption {
	return func(c *config) {
		c.logger = logger
	}
}

// WithPrioritySampling is deprecated, and priority sampling is enabled by default.
// When using distributed tracing, the priority sampling value is propagated in order to
// get all the parts of a distributed trace sampled.
// To learn more about priority sampling, please visit:
// https://docs.datadoghq.com/tracing/getting_further/trace_sampling_and_storage/#priority-sampling-for-distributed-tracing
func WithPrioritySampling() StartOption {
	return func(c *config) {
		// This is now enabled by default.
	}
}

// WithDebugMode enables debug mode on the tracer, resulting in more verbose logging.
func WithDebugMode(enabled bool) StartOption {
	return func(c *config) {
		c.debug = enabled
	}
}

// WithPropagator sets an alternative propagator to be used by the tracer.
func WithPropagator(p Propagator) StartOption {
	return func(c *config) {
		c.propagator = p
	}
}

// WithServiceName sets the default service name to be used with the tracer.
func WithServiceName(name string) StartOption {
	return func(c *config) {
		c.serviceName = name
	}
}

// WithAgentAddr sets the address where the agent is located. The default is
// localhost:8126. It should contain both host and port.
func WithAgentAddr(addr string) StartOption {
	return func(c *config) {
		c.agentAddr = addr
	}
}

// WithEnv sets the environment to which all traces started by the tracer will be submitted.
// The default value is the environment variable DD_ENV, if it is set.
func WithEnv(env string) StartOption {
	return WithGlobalTag(ext.Environment, env)
}

// WithGlobalTag sets a key/value pair which will be set as a tag on all spans
// created by tracer. This option may be used multiple times.
func WithGlobalTag(k string, v interface{}) StartOption {
	return func(c *config) {
		if c.globalTags == nil {
			c.globalTags = make(map[string]interface{})
		}
		c.globalTags[k] = v
	}
}

// WithSampler sets the given sampler to be used with the tracer. By default
// an all-permissive sampler is used.
func WithSampler(s Sampler) StartOption {
	return func(c *config) {
		c.sampler = s
	}
}

// WithHTTPRoundTripper allows customizing the underlying HTTP transport for
// emitting spans. This is useful for advanced customization such as emitting
// spans to a unix domain socket. The default should be used in most cases.
func WithHTTPRoundTripper(r http.RoundTripper) StartOption {
	return func(c *config) {
		c.httpRoundTripper = r
	}
}

// WithAnalytics allows specifying whether Trace Search & Analytics should be enabled
// for integrations.
func WithAnalytics(on bool) StartOption {
	return func(cfg *config) {
		if on {
			globalconfig.SetAnalyticsRate(1.0)
		} else {
			globalconfig.SetAnalyticsRate(math.NaN())
		}
	}
}

// WithAnalyticsRate sets the global sampling rate for sampling APM events.
func WithAnalyticsRate(rate float64) StartOption {
	return func(_ *config) {
		if rate >= 0.0 && rate <= 1.0 {
			globalconfig.SetAnalyticsRate(rate)
		} else {
			globalconfig.SetAnalyticsRate(math.NaN())
		}
	}
}

// WithRuntimeMetrics enables automatic collection of runtime metrics every 10 seconds.
func WithRuntimeMetrics() StartOption {
	return func(cfg *config) {
		cfg.runtimeMetrics = true
	}
}

// WithDogstatsdAddress specifies the address to connect to for sending metrics
// to the Datadog Agent. If not set, it defaults to "localhost:8125" or to the
// combination of the environment variables DD_AGENT_HOST and DD_DOGSTATSD_PORT.
// This option is in effect when WithRuntimeMetrics is enabled.
func WithDogstatsdAddress(addr string) StartOption {
	return func(cfg *config) {
		cfg.dogstatsdAddr = addr
	}
}

// WithSamplingRules specifies the sampling rates to apply to spans based on the
// provided rules.
func WithSamplingRules(rules []SamplingRule) StartOption {
	return func(cfg *config) {
		cfg.samplingRules = rules
	}
}

// StartSpanOption is a configuration option for StartSpan. It is aliased in order
// to help godoc group all the functions returning it together. It is considered
// more correct to refer to it as the type as the origin, ddtrace.StartSpanOption.
type StartSpanOption = ddtrace.StartSpanOption

// Tag sets the given key/value pair as a tag on the started Span.
func Tag(k string, v interface{}) StartSpanOption {
	return func(cfg *ddtrace.StartSpanConfig) {
		if cfg.Tags == nil {
			cfg.Tags = map[string]interface{}{}
		}
		cfg.Tags[k] = v
	}
}

// ServiceName sets the given service name on the started span. For example "http.server".
func ServiceName(name string) StartSpanOption {
	return Tag(ext.ServiceName, name)
}

// ResourceName sets the given resource name on the started span. A resource could
// be an SQL query, a URL, an RPC method or something else.
func ResourceName(name string) StartSpanOption {
	return Tag(ext.ResourceName, name)
}

// SpanType sets the given span type on the started span. Some examples in the case of
// the Datadog APM product could be "web", "db" or "cache".
func SpanType(name string) StartSpanOption {
	return Tag(ext.SpanType, name)
}

// WithSpanID sets the SpanID on the started span, instead of using a random number.
// If there is no parent Span (eg from ChildOf), then the TraceID will also be set to the
// value given here.
func WithSpanID(id uint64) StartSpanOption {
	return func(cfg *ddtrace.StartSpanConfig) {
		cfg.SpanID = id
	}
}

// ChildOf tells StartSpan to use the given span context as a parent for the
// created span.
func ChildOf(ctx ddtrace.SpanContext) StartSpanOption {
	return func(cfg *ddtrace.StartSpanConfig) {
		cfg.Parent = ctx
	}
}

// StartTime sets a custom time as the start time for the created span. By
// default a span is started using the creation time.
func StartTime(t time.Time) StartSpanOption {
	return func(cfg *ddtrace.StartSpanConfig) {
		cfg.StartTime = t
	}
}

// FinishOption is a configuration option for FinishSpan. It is aliased in order
// to help godoc group all the functions returning it together. It is considered
// more correct to refer to it as the type as the origin, ddtrace.FinishOption.
type FinishOption = ddtrace.FinishOption

// FinishTime sets the given time as the finishing time for the span. By default,
// the current time is used.
func FinishTime(t time.Time) FinishOption {
	return func(cfg *ddtrace.FinishConfig) {
		cfg.FinishTime = t
	}
}

// WithError marks the span as having had an error. It uses the information from
// err to set tags such as the error message, error type and stack trace. It has
// no effect if the error is nil.
func WithError(err error) FinishOption {
	return func(cfg *ddtrace.FinishConfig) {
		cfg.Error = err
	}
}

// NoDebugStack prevents any error presented using the WithError finishing option
// from generating a stack trace. This is useful in situations where errors are frequent
// and performance is critical.
func NoDebugStack() FinishOption {
	return func(cfg *ddtrace.FinishConfig) {
		cfg.NoDebugStack = true
	}
}

// StackFrames limits the number of stack frames included into erroneous spans to n, starting from skip.
func StackFrames(n, skip uint) FinishOption {
	if n == 0 {
		return NoDebugStack()
	}
	return func(cfg *ddtrace.FinishConfig) {
		cfg.StackFrames = n
		cfg.SkipStackFrames = skip
	}
}
