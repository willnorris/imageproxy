// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package tracer

import (
	"os"
	"strconv"
	"sync"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/internal"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"

	"github.com/DataDog/datadog-go/statsd"
)

var _ ddtrace.Tracer = (*tracer)(nil)

// tracer creates, buffers and submits Spans which are used to time blocks of
// computation. They are accumulated and streamed into an internal payload,
// which is flushed to the agent whenever its size exceeds a specific threshold
// or when a certain interval of time has passed, whichever happens first.
//
// tracer operates based on a worker loop which responds to various request
// channels. It additionally holds two buffers which accumulates error and trace
// queues to be processed by the payload encoder.
type tracer struct {
	*config
	*payload

	// payloadChan receives traces to be added to the payload.
	payloadChan chan []*span

	// climit limits the number of concurrent outgoing connections
	climit chan struct{}

	// stop causes the tracer to shut down when closed.
	stop chan struct{}

	// stopOnce ensures the tracer is stopped exactly once.
	stopOnce sync.Once

	// wg waits for all goroutines to exit when stopping.
	wg sync.WaitGroup

	// prioritySampling holds an instance of the priority sampler.
	prioritySampling *prioritySampler

	// pid of the process
	pid string

	// These integers track metrics about spans and traces as they are started,
	// finished, and dropped
	spansStarted, spansFinished, tracesDropped int64

	// rulesSampling holds an instance of the rules sampler. These are user-defined
	// rules for applying a sampling rate to spans that match the designated service
	// or operation name.
	rulesSampling *rulesSampler
}

const (
	// flushInterval is the interval at which the payload contents will be flushed
	// to the transport.
	flushInterval = 2 * time.Second

	// payloadMaxLimit is the maximum payload size allowed and should indicate the
	// maximum size of the package that the agent can receive.
	payloadMaxLimit = 9.5 * 1024 * 1024 // 9.5 MB

	// payloadSizeLimit specifies the maximum allowed size of the payload before
	// it will trigger a flush to the transport.
	payloadSizeLimit = payloadMaxLimit / 2

	// concurrentConnectionLimit specifies the maximum number of concurrent outgoing
	// connections allowed.
	concurrentConnectionLimit = 100
)

// statsInterval is the interval at which health metrics will be sent with the
// statsd client; replaced in tests.
var statsInterval = 10 * time.Second

// Start starts the tracer with the given set of options. It will stop and replace
// any running tracer, meaning that calling it several times will result in a restart
// of the tracer by replacing the current instance with a new one.
func Start(opts ...StartOption) {
	if internal.Testing {
		return // mock tracer active
	}
	internal.SetGlobalTracer(newTracer(opts...))
}

// Stop stops the started tracer. Subsequent calls are valid but become no-op.
func Stop() {
	internal.SetGlobalTracer(&internal.NoopTracer{})
	log.Flush()
}

// Span is an alias for ddtrace.Span. It is here to allow godoc to group methods returning
// ddtrace.Span. It is recommended and is considered more correct to refer to this type as
// ddtrace.Span instead.
type Span = ddtrace.Span

// StartSpan starts a new span with the given operation name and set of options.
// If the tracer is not started, calling this function is a no-op.
func StartSpan(operationName string, opts ...StartSpanOption) Span {
	return internal.GetGlobalTracer().StartSpan(operationName, opts...)
}

// Extract extracts a SpanContext from the carrier. The carrier is expected
// to implement TextMapReader, otherwise an error is returned.
// If the tracer is not started, calling this function is a no-op.
func Extract(carrier interface{}) (ddtrace.SpanContext, error) {
	return internal.GetGlobalTracer().Extract(carrier)
}

// Inject injects the given SpanContext into the carrier. The carrier is
// expected to implement TextMapWriter, otherwise an error is returned.
// If the tracer is not started, calling this function is a no-op.
func Inject(ctx ddtrace.SpanContext, carrier interface{}) error {
	return internal.GetGlobalTracer().Inject(ctx, carrier)
}

// payloadQueueSize is the buffer size of the trace channel.
const payloadQueueSize = 1000

func newUnstartedTracer(opts ...StartOption) *tracer {
	c := new(config)
	defaults(c)
	for _, fn := range opts {
		fn(c)
	}
	if c.transport == nil {
		c.transport = newTransport(c.agentAddr, c.httpRoundTripper)
	}
	if c.propagator == nil {
		c.propagator = NewPropagator(nil)
	}
	if c.logger != nil {
		log.UseLogger(c.logger)
	}
	if c.debug {
		log.SetLevel(log.LevelDebug)
	}
	if c.statsd == nil {
		client, err := statsd.New(c.dogstatsdAddr, statsd.WithMaxMessagesPerPayload(40), statsd.WithTags(statsTags(c)))
		if err != nil {
			log.Warn("Runtime and health metrics disabled: %v", err)
			c.statsd = &statsd.NoOpClient{}
		} else {
			c.statsd = client
		}
	}
	return &tracer{
		config:           c,
		payload:          newPayload(),
		payloadChan:      make(chan []*span, payloadQueueSize),
		stop:             make(chan struct{}),
		rulesSampling:    newRulesSampler(c.samplingRules),
		climit:           make(chan struct{}, concurrentConnectionLimit),
		prioritySampling: newPrioritySampler(),
		pid:              strconv.Itoa(os.Getpid()),
	}
}

func newTracer(opts ...StartOption) *tracer {
	t := newUnstartedTracer(opts...)
	c := t.config
	t.config.statsd.Incr("datadog.tracer.started", nil, 1)
	if c.runtimeMetrics {
		log.Debug("Runtime metrics enabled.")
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			t.reportRuntimeMetrics(defaultMetricsReportInterval)
		}()
	}
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		tick := t.config.tickChan
		if tick == nil {
			ticker := time.NewTicker(flushInterval)
			defer ticker.Stop()
			tick = ticker.C
		}
		t.worker(tick)
	}()

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.reportHealthMetrics(statsInterval)
	}()
	return t
}

// worker receives finished traces to be added into the payload, as well
// as periodically flushes traces to the transport.
func (t *tracer) worker(tick <-chan time.Time) {
	defer t.config.statsd.Close()

	for {
		select {
		case trace := <-t.payloadChan:
			t.pushPayload(trace)

		case <-tick:
			t.config.statsd.Incr("datadog.tracer.flush_triggered", []string{"reason:scheduled"}, 1)
			t.flush()

		case <-t.stop:
		loop:
			// the loop ensures that the payload channel is fully drained
			// before the final flush to ensure no traces are lost (see #526)
			for {
				select {
				case trace := <-t.payloadChan:
					t.pushPayload(trace)
				default:
					break loop
				}
			}
			t.config.statsd.Incr("datadog.tracer.flush_triggered", []string{"reason:shutdown"}, 1)
			t.flush()
			t.config.statsd.Incr("datadog.tracer.stopped", nil, 1)
			return
		}
	}
}

func (t *tracer) pushTrace(trace []*span) {
	select {
	case <-t.stop:
		return
	default:
	}
	select {
	case t.payloadChan <- trace:
	default:
		log.Error("payload queue full, dropping %d traces", len(trace))
	}
}

// StartSpan creates, starts, and returns a new Span with the given `operationName`.
func (t *tracer) StartSpan(operationName string, options ...ddtrace.StartSpanOption) ddtrace.Span {
	var opts ddtrace.StartSpanConfig
	for _, fn := range options {
		fn(&opts)
	}
	var startTime int64
	if opts.StartTime.IsZero() {
		startTime = now()
	} else {
		startTime = opts.StartTime.UnixNano()
	}
	var context *spanContext
	if opts.Parent != nil {
		if ctx, ok := opts.Parent.(*spanContext); ok {
			context = ctx
		}
	}
	id := opts.SpanID
	if id == 0 {
		id = random.Uint64()
	}
	// span defaults
	span := &span{
		Name:     operationName,
		Service:  t.config.serviceName,
		Resource: operationName,
		SpanID:   id,
		TraceID:  id,
		Start:    startTime,
		taskEnd:  startExecutionTracerTask(operationName),
	}
	if context != nil {
		// this is a child span
		span.TraceID = context.traceID
		span.ParentID = context.spanID
		if p, ok := context.samplingPriority(); ok {
			span.setMetric(keySamplingPriority, float64(p))
		}
		if context.span != nil {
			// local parent, inherit service
			context.span.RLock()
			span.Service = context.span.Service
			context.span.RUnlock()
		} else {
			// remote parent
			if context.origin != "" {
				// mark origin
				span.setMeta(keyOrigin, context.origin)
			}
		}
	}
	span.context = newSpanContext(span, context)
	if context == nil || context.span == nil {
		// this is either a root span or it has a remote parent, we should add the PID.
		span.setMeta(ext.Pid, t.pid)
		if t.hostname != "" {
			span.setMeta(keyHostname, t.hostname)
		}
		if _, ok := opts.Tags[ext.ServiceName]; !ok && t.config.runtimeMetrics {
			// this is a root span in the global service; runtime metrics should
			// be linked to it:
			span.setMeta("language", "go")
		}
	}
	// add tags from options
	for k, v := range opts.Tags {
		span.SetTag(k, v)
	}
	// add global tags
	for k, v := range t.config.globalTags {
		span.SetTag(k, v)
	}
	if context == nil {
		// this is a brand new trace, sample it
		t.sample(span)
	}
	return span
}

// Stop stops the tracer.
func (t *tracer) Stop() {
	t.stopOnce.Do(func() {
		close(t.stop)
	})
	t.wg.Wait()
}

// Inject uses the configured or default TextMap Propagator.
func (t *tracer) Inject(ctx ddtrace.SpanContext, carrier interface{}) error {
	return t.config.propagator.Inject(ctx, carrier)
}

// Extract uses the configured or default TextMap Propagator.
func (t *tracer) Extract(carrier interface{}) (ddtrace.SpanContext, error) {
	return t.config.propagator.Extract(carrier)
}

// flush will push any currently buffered traces to the server.
func (t *tracer) flush() {
	if t.payload.itemCount() == 0 {
		return
	}
	t.wg.Add(1)
	t.climit <- struct{}{}
	go func(p *payload) {
		defer func(start time.Time) {
			<-t.climit
			t.wg.Done()
			t.config.statsd.Timing("datadog.tracer.flush_duration", time.Since(start), nil, 1)
		}(time.Now())
		size, count := p.size(), p.itemCount()
		log.Debug("Sending payload: size: %d traces: %d\n", size, count)
		rc, err := t.config.transport.send(p)
		if err != nil {
			t.config.statsd.Count("datadog.tracer.traces_dropped", int64(count), []string{"reason:send_failed"}, 1)
			log.Error("lost %d traces: %v", count, err)
		} else {
			t.config.statsd.Count("datadog.tracer.flush_bytes", int64(size), nil, 1)
			t.config.statsd.Count("datadog.tracer.flush_traces", int64(count), nil, 1)
			if err := t.prioritySampling.readRatesJSON(rc); err != nil {
				t.config.statsd.Incr("datadog.tracer.decode_error", nil, 1)
			}
		}
	}(t.payload)
	t.payload = newPayload()
}

// pushPayload pushes the trace onto the payload. If the payload becomes
// larger than the threshold as a result, it sends a flush request.
func (t *tracer) pushPayload(trace []*span) {
	if err := t.payload.push(trace); err != nil {
		t.config.statsd.Incr("datadog.tracer.traces_dropped", []string{"reason:encoding_error"}, 1)
		log.Error("error encoding msgpack: %v", err)
	}
	if t.payload.size() > payloadSizeLimit {
		t.config.statsd.Incr("datadog.tracer.flush_triggered", []string{"reason:size"}, 1)
		t.flush()
	}
}

// sampleRateMetricKey is the metric key holding the applied sample rate. Has to be the same as the Agent.
const sampleRateMetricKey = "_sample_rate"

// Sample samples a span with the internal sampler.
func (t *tracer) sample(span *span) {
	if _, ok := span.context.samplingPriority(); ok {
		// sampling decision was already made
		return
	}
	sampler := t.config.sampler
	if !sampler.Sample(span) {
		span.context.drop = true
		return
	}
	if rs, ok := sampler.(RateSampler); ok && rs.Rate() < 1 {
		span.setMetric(sampleRateMetricKey, rs.Rate())
	}
	if t.rulesSampling.apply(span) {
		return
	}
	t.prioritySampling.apply(span)
}
