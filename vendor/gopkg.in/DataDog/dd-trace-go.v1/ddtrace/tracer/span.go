// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

//go:generate msgp -unexported -marshal=false -o=span_msgp.go -tests=false

package tracer

import (
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"

	"github.com/tinylib/msgp/msgp"
	"golang.org/x/xerrors"
)

type (
	// spanList implements msgp.Encodable on top of a slice of spans.
	spanList []*span

	// spanLists implements msgp.Decodable on top of a slice of spanList.
	// This type is only used in tests.
	spanLists []spanList
)

var (
	_ ddtrace.Span   = (*span)(nil)
	_ msgp.Encodable = (*spanList)(nil)
	_ msgp.Decodable = (*spanLists)(nil)
)

// errorConfig holds customization options for setting error tags.
type errorConfig struct {
	noDebugStack bool
	stackFrames  uint
	stackSkip    uint
}

// span represents a computation. Callers must call Finish when a span is
// complete to ensure it's submitted.
type span struct {
	sync.RWMutex `msg:"-"`

	Name     string             `msg:"name"`              // operation name
	Service  string             `msg:"service"`           // service name (i.e. "grpc.server", "http.request")
	Resource string             `msg:"resource"`          // resource name (i.e. "/user?id=123", "SELECT * FROM users")
	Type     string             `msg:"type"`              // protocol associated with the span (i.e. "web", "db", "cache")
	Start    int64              `msg:"start"`             // span start time expressed in nanoseconds since epoch
	Duration int64              `msg:"duration"`          // duration of the span expressed in nanoseconds
	Meta     map[string]string  `msg:"meta,omitempty"`    // arbitrary map of metadata
	Metrics  map[string]float64 `msg:"metrics,omitempty"` // arbitrary map of numeric metrics
	SpanID   uint64             `msg:"span_id"`           // identifier of this span
	TraceID  uint64             `msg:"trace_id"`          // identifier of the root span
	ParentID uint64             `msg:"parent_id"`         // identifier of the span's direct parent
	Error    int32              `msg:"error"`             // error status of the span; 0 means no errors

	finished bool         `msg:"-"` // true if the span has been submitted to a tracer.
	context  *spanContext `msg:"-"` // span propagation context
	taskEnd  func()       // ends execution tracer (runtime/trace) task, if started
}

// Context yields the SpanContext for this Span. Note that the return
// value of Context() is still valid after a call to Finish(). This is
// called the span context and it is different from Go's context.
func (s *span) Context() ddtrace.SpanContext { return s.context }

// SetBaggageItem sets a key/value pair as baggage on the span. Baggage items
// are propagated down to descendant spans and injected cross-process. Use with
// care as it adds extra load onto your tracing layer.
func (s *span) SetBaggageItem(key, val string) {
	s.context.setBaggageItem(key, val)
}

// BaggageItem gets the value for a baggage item given its key. Returns the
// empty string if the value isn't found in this Span.
func (s *span) BaggageItem(key string) string {
	return s.context.baggageItem(key)
}

// SetTag adds a set of key/value metadata to the span.
func (s *span) SetTag(key string, value interface{}) {
	s.Lock()
	defer s.Unlock()
	// We don't lock spans when flushing, so we could have a data race when
	// modifying a span as it's being flushed. This protects us against that
	// race, since spans are marked `finished` before we flush them.
	if s.finished {
		return
	}
	switch key {
	case ext.Error:
		s.setTagError(value, &errorConfig{})
		return
	}
	if v, ok := value.(bool); ok {
		s.setTagBool(key, v)
		return
	}
	if v, ok := value.(string); ok {
		s.setMeta(key, v)
		return
	}
	if v, ok := toFloat64(value); ok {
		s.setMetric(key, v)
		return
	}
	// not numeric, not a string, not a bool, and not an error
	s.setMeta(key, fmt.Sprint(value))
}

// setTagError sets the error tag. It accounts for various valid scenarios.
// This method is not safe for concurrent use.
func (s *span) setTagError(value interface{}, cfg *errorConfig) {
	if s.finished {
		return
	}
	switch v := value.(type) {
	case bool:
		// bool value as per Opentracing spec.
		if !v {
			s.Error = 0
		} else {
			s.Error = 1
		}
	case error:
		// if anyone sets an error value as the tag, be nice here
		// and provide all the benefits.
		s.Error = 1
		s.setMeta(ext.ErrorMsg, v.Error())
		s.setMeta(ext.ErrorType, reflect.TypeOf(v).String())
		if !cfg.noDebugStack {
			if cfg.stackFrames == 0 {
				s.setMeta(ext.ErrorStack, string(debug.Stack()))
			} else {
				s.setMeta(ext.ErrorStack, takeStacktrace(cfg.stackFrames, cfg.stackSkip))
			}
		}
		switch v.(type) {
		case xerrors.Formatter:
			s.setMeta(ext.ErrorDetails, fmt.Sprintf("%+v", v))
		case fmt.Formatter:
			// pkg/errors approach
			s.setMeta(ext.ErrorDetails, fmt.Sprintf("%+v", v))
		}
	case nil:
		// no error
		s.Error = 0
	default:
		// in all other cases, let's assume that setting this tag
		// is the result of an error.
		s.Error = 1
	}
}

// takeStacktrace takes stacktrace
func takeStacktrace(n, skip uint) string {
	var builder strings.Builder
	pcs := make([]uintptr, n)

	// +2 to exclude runtime.Callers and takeStacktrace
	numFrames := runtime.Callers(2+int(skip), pcs)
	if numFrames == 0 {
		return ""
	}
	frames := runtime.CallersFrames(pcs[:numFrames])
	for i := 0; ; i++ {
		frame, more := frames.Next()
		if i != 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(frame.Function)
		builder.WriteByte('\n')
		builder.WriteByte('\t')
		builder.WriteString(frame.File)
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(frame.Line))
		if !more {
			break
		}
	}
	return builder.String()
}

// setMeta sets a string tag. This method is not safe for concurrent use.
func (s *span) setMeta(key, v string) {
	if s.Meta == nil {
		s.Meta = make(map[string]string, 1)
	}
	switch key {
	case ext.SpanName:
		s.Name = v
	case ext.ServiceName:
		s.Service = v
	case ext.ResourceName:
		s.Resource = v
	case ext.SpanType:
		s.Type = v
	default:
		s.Meta[key] = v
	}
}

// setTagBool sets a boolean tag on the span.
func (s *span) setTagBool(key string, v bool) {
	switch key {
	case ext.AnalyticsEvent:
		if v {
			s.setMetric(ext.EventSampleRate, 1.0)
		} else {
			s.setMetric(ext.EventSampleRate, 0.0)
		}
	case ext.ManualDrop:
		if v {
			s.setMetric(ext.SamplingPriority, ext.PriorityUserReject)
		}
	case ext.ManualKeep:
		if v {
			s.setMetric(ext.SamplingPriority, ext.PriorityUserKeep)
		}
	default:
		if v {
			s.setMeta(key, "true")
		} else {
			s.setMeta(key, "false")
		}
	}
}

// setMetric sets a numeric tag, in our case called a metric. This method
// is not safe for concurrent use.
func (s *span) setMetric(key string, v float64) {
	if s.Metrics == nil {
		s.Metrics = make(map[string]float64, 1)
	}
	switch key {
	case ext.SamplingPriority:
		// setting sampling priority per spec
		s.Metrics[keySamplingPriority] = v
		s.context.setSamplingPriority(int(v))
	default:
		s.Metrics[key] = v
	}
}

// Finish closes this Span (but not its children) providing the duration
// of its part of the tracing session.
func (s *span) Finish(opts ...ddtrace.FinishOption) {
	t := now()
	if len(opts) > 0 {
		var cfg ddtrace.FinishConfig
		for _, fn := range opts {
			fn(&cfg)
		}
		if !cfg.FinishTime.IsZero() {
			t = cfg.FinishTime.UnixNano()
		}
		if cfg.Error != nil {
			s.Lock()
			s.setTagError(cfg.Error, &errorConfig{
				noDebugStack: cfg.NoDebugStack,
				stackFrames:  cfg.StackFrames,
				stackSkip:    cfg.SkipStackFrames,
			})
			s.Unlock()
		}
	}
	if s.taskEnd != nil {
		s.taskEnd()
	}
	s.finish(t)
}

// SetOperationName sets or changes the operation name.
func (s *span) SetOperationName(operationName string) {
	s.Lock()
	defer s.Unlock()

	s.Name = operationName
}

func (s *span) finish(finishTime int64) {
	s.Lock()
	defer s.Unlock()
	// We don't lock spans when flushing, so we could have a data race when
	// modifying a span as it's being flushed. This protects us against that
	// race, since spans are marked `finished` before we flush them.
	if s.finished {
		// already finished
		return
	}
	if s.Duration == 0 {
		s.Duration = finishTime - s.Start
	}
	s.finished = true

	if s.context.drop {
		// not sampled by local sampler
		return
	}
	s.context.finish()
}

// String returns a human readable representation of the span. Not for
// production, just debugging.
func (s *span) String() string {
	lines := []string{
		fmt.Sprintf("Name: %s", s.Name),
		fmt.Sprintf("Service: %s", s.Service),
		fmt.Sprintf("Resource: %s", s.Resource),
		fmt.Sprintf("TraceID: %d", s.TraceID),
		fmt.Sprintf("SpanID: %d", s.SpanID),
		fmt.Sprintf("ParentID: %d", s.ParentID),
		fmt.Sprintf("Start: %s", time.Unix(0, s.Start)),
		fmt.Sprintf("Duration: %s", time.Duration(s.Duration)),
		fmt.Sprintf("Error: %d", s.Error),
		fmt.Sprintf("Type: %s", s.Type),
		"Tags:",
	}
	s.RLock()
	for key, val := range s.Meta {
		lines = append(lines, fmt.Sprintf("\t%s:%s", key, val))
	}
	for key, val := range s.Metrics {
		lines = append(lines, fmt.Sprintf("\t%s:%f", key, val))
	}
	s.RUnlock()
	return strings.Join(lines, "\n")
}

const (
	keySamplingPriority        = "_sampling_priority_v1"
	keySamplingPriorityRate    = "_sampling_priority_rate_v1"
	keyOrigin                  = "_dd.origin"
	keyHostname                = "_dd.hostname"
	keyRulesSamplerAppliedRate = "_dd.rule_psr"
	keyRulesSamplerLimiterRate = "_dd.limit_psr"
)
