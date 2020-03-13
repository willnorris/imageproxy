// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package tracer

import (
	"encoding/json"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"

	"golang.org/x/time/rate"
)

// Sampler is the generic interface of any sampler. It must be safe for concurrent use.
type Sampler interface {
	// Sample returns true if the given span should be sampled.
	Sample(span Span) bool
}

// RateSampler is a sampler implementation which randomly selects spans using a
// provided rate. For example, a rate of 0.75 will permit 75% of the spans.
// RateSampler implementations should be safe for concurrent use.
type RateSampler interface {
	Sampler

	// Rate returns the current sample rate.
	Rate() float64

	// SetRate sets a new sample rate.
	SetRate(rate float64)
}

// rateSampler samples from a sample rate.
type rateSampler struct {
	sync.RWMutex
	rate float64
}

// NewAllSampler is a short-hand for NewRateSampler(1). It is all-permissive.
func NewAllSampler() RateSampler { return NewRateSampler(1) }

// NewRateSampler returns an initialized RateSampler with a given sample rate.
func NewRateSampler(rate float64) RateSampler {
	return &rateSampler{rate: rate}
}

// Rate returns the current rate of the sampler.
func (r *rateSampler) Rate() float64 {
	r.RLock()
	defer r.RUnlock()
	return r.rate
}

// SetRate sets a new sampling rate.
func (r *rateSampler) SetRate(rate float64) {
	r.Lock()
	r.rate = rate
	r.Unlock()
}

// constants used for the Knuth hashing, same as agent.
const knuthFactor = uint64(1111111111111111111)

// Sample returns true if the given span should be sampled.
func (r *rateSampler) Sample(spn ddtrace.Span) bool {
	if r.rate == 1 {
		// fast path
		return true
	}
	s, ok := spn.(*span)
	if !ok {
		return false
	}
	r.RLock()
	defer r.RUnlock()
	return sampledByRate(s.TraceID, r.rate)
}

// sampledByRate verifies if the number n should be sampled at the specified
// rate.
func sampledByRate(n uint64, rate float64) bool {
	if rate < 1 {
		return n*knuthFactor < uint64(rate*math.MaxUint64)
	}
	return true
}

// prioritySampler holds a set of per-service sampling rates and applies
// them to spans.
type prioritySampler struct {
	mu          sync.RWMutex
	rates       map[string]float64
	defaultRate float64
}

func newPrioritySampler() *prioritySampler {
	return &prioritySampler{
		rates:       make(map[string]float64),
		defaultRate: 1.,
	}
}

// readRatesJSON will try to read the rates as JSON from the given io.ReadCloser.
func (ps *prioritySampler) readRatesJSON(rc io.ReadCloser) error {
	var payload struct {
		Rates map[string]float64 `json:"rate_by_service"`
	}
	if err := json.NewDecoder(rc).Decode(&payload); err != nil {
		return err
	}
	rc.Close()
	const defaultRateKey = "service:,env:"
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.rates = payload.Rates
	if v, ok := ps.rates[defaultRateKey]; ok {
		ps.defaultRate = v
		delete(ps.rates, defaultRateKey)
	}
	return nil
}

// getRate returns the sampling rate to be used for the given span. Callers must
// guard the span.
func (ps *prioritySampler) getRate(spn *span) float64 {
	key := "service:" + spn.Service + ",env:" + spn.Meta[ext.Environment]
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if rate, ok := ps.rates[key]; ok {
		return rate
	}
	return ps.defaultRate
}

// apply applies sampling priority to the given span. Caller must ensure it is safe
// to modify the span.
func (ps *prioritySampler) apply(spn *span) {
	rate := ps.getRate(spn)
	if sampledByRate(spn.TraceID, rate) {
		spn.SetTag(ext.SamplingPriority, ext.PriorityAutoKeep)
	} else {
		spn.SetTag(ext.SamplingPriority, ext.PriorityAutoReject)
	}
	spn.SetTag(keySamplingPriorityRate, rate)
}

// rulesSampler allows a user-defined list of rules to apply to spans.
// These rules can match based on the span's Service, Name or both.
// When making a sampling decision, the rules are checked in order until
// a match is found.
// If a match is found, the rate from that rule is used.
// If no match is found, and the DD_TRACE_SAMPLE_RATE environment variable
// was set to a valid rate, that value is used.
// Otherwise, the rules sampler didn't apply to the span, and the decision
// is passed to the priority sampler.
//
// The rate is used to determine if the span should be sampled, but an upper
// limit can be defined using the DD_TRACE_RATE_LIMIT environment variable.
// Its value is the number of spans to sample per second.
// Spans that matched the rules but exceeded the rate limit are not sampled.
type rulesSampler struct {
	rules      []SamplingRule // the rules to match spans with
	globalRate float64        // a rate to apply when no rules match a span
	limiter    *rateLimiter   // used to limit the volume of spans sampled
}

// newRulesSampler configures a *rulesSampler instance using the given set of rules.
// Invalid rules or environment variable values are tolerated, by logging warnings and then ignoring them.
func newRulesSampler(rules []SamplingRule) *rulesSampler {
	return &rulesSampler{
		rules:      appliedSamplingRules(rules),
		globalRate: globalSampleRate(),
		limiter:    newRateLimiter(),
	}
}

// appliedSamplingRules validates the user-provided rules and returns an internal representation.
// If the DD_TRACE_SAMPLING_RULES environment variable is set, it will replace the given rules.
func appliedSamplingRules(rules []SamplingRule) []SamplingRule {
	rulesFromEnv := os.Getenv("DD_TRACE_SAMPLING_RULES")
	if rulesFromEnv != "" {
		rules = rules[:0]
		jsonRules := []struct {
			Service string      `json:"service"`
			Name    string      `json:"name"`
			Rate    json.Number `json:"sample_rate"`
		}{}
		err := json.Unmarshal([]byte(rulesFromEnv), &jsonRules)
		if err != nil {
			log.Warn("error parsing DD_TRACE_SAMPLING_RULES: %v", err)
			return nil
		}
		for _, v := range jsonRules {
			if v.Rate == "" {
				log.Warn("error parsing rule: rate not provided")
				continue
			}
			rate, err := v.Rate.Float64()
			if err != nil {
				log.Warn("error parsing rule: invalid rate: %v", err)
				continue
			}
			switch {
			case v.Service != "" && v.Name != "":
				rules = append(rules, NameServiceRule(v.Name, v.Service, rate))
			case v.Service != "":
				rules = append(rules, ServiceRule(v.Service, rate))
			case v.Name != "":
				rules = append(rules, NameRule(v.Name, rate))
			}
		}
	}
	validRules := make([]SamplingRule, 0, len(rules))
	for _, v := range rules {
		if !(v.Rate >= 0.0 && v.Rate <= 1.0) {
			log.Warn("ignoring rule %+v: rate is out of range", v)
			continue
		}
		validRules = append(validRules, v)
	}
	return validRules
}

// globalSampleRate returns the sampling rate found in the DD_TRACE_SAMPLE_RATE environment variable.
// If it is invalid or not within the 0-1 range, NaN is returned.
func globalSampleRate() float64 {
	defaultRate := math.NaN()
	v := os.Getenv("DD_TRACE_SAMPLE_RATE")
	if v == "" {
		return defaultRate
	}
	r, err := strconv.ParseFloat(v, 64)
	if err != nil {
		log.Warn("ignoring DD_TRACE_SAMPLE_RATE: error: %v", err)
		return defaultRate
	}
	if r >= 0.0 && r <= 1.0 {
		return r
	}
	log.Warn("ignoring DD_TRACE_SAMPLE_RATE: out of range %f", r)
	return defaultRate
}

// defaultRateLimit specifies the default trace rate limit used when DD_TRACE_RATE_LIMIT is not set.
const defaultRateLimit = 100.0

// newRateLimiter returns a rate limiter which restricts the number of traces sampled per second.
// This defaults to 100.0. The DD_TRACE_RATE_LIMIT environment variable may override the default.
func newRateLimiter() *rateLimiter {
	limit := defaultRateLimit
	v := os.Getenv("DD_TRACE_RATE_LIMIT")
	if v != "" {
		l, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.Warn("using default rate limit because DD_TRACE_RATE_LIMIT is invalid: %v", err)
		} else if l < 0.0 {
			log.Warn("using default rate limit because DD_TRACE_RATE_LIMIT is negative: %f", l)
		} else {
			// override the default limit
			limit = l
		}
	}
	return &rateLimiter{
		limiter:  rate.NewLimiter(rate.Limit(limit), int(math.Ceil(limit))),
		prevTime: time.Now(),
	}
}

// apply uses the sampling rules to determine the sampling rate for the
// provided span. If the rules don't match, and a default rate hasn't been
// set using DD_TRACE_SAMPLE_RATE, then it returns false and the span is not
// modified.
func (rs *rulesSampler) apply(span *span) bool {
	if len(rs.rules) == 0 && math.IsNaN(rs.globalRate) {
		// short path when disabled
		return false
	}

	var matched bool
	rate := rs.globalRate
	for _, rule := range rs.rules {
		if rule.match(span) {
			matched = true
			rate = rule.Rate
			break
		}
	}
	if !matched && math.IsNaN(rate) {
		// no matching rule or global rate, so we want to fall back
		// to priority sampling
		return false
	}

	rs.applyRate(span, rate, time.Now())
	return true
}

func (rs *rulesSampler) applyRate(span *span, rate float64, now time.Time) {
	span.SetTag(keyRulesSamplerAppliedRate, rate)
	if !sampledByRate(span.TraceID, rate) {
		span.SetTag(ext.SamplingPriority, ext.PriorityAutoReject)
		return
	}

	sampled, rate := rs.limiter.allowOne(now)
	if sampled {
		span.SetTag(ext.SamplingPriority, ext.PriorityAutoKeep)
	} else {
		span.SetTag(ext.SamplingPriority, ext.PriorityAutoReject)
	}
	span.SetTag(keyRulesSamplerLimiterRate, rate)
}

// SamplingRule is used for applying sampling rates to spans that match
// the service name, operation name or both.
// For basic usage, consider using the helper functions ServiceRule, NameRule, etc.
type SamplingRule struct {
	Service *regexp.Regexp
	Name    *regexp.Regexp
	Rate    float64

	exactService string
	exactName    string
}

// ServiceRule returns a SamplingRule that applies the provided sampling rate
// to spans that match the service name provided.
func ServiceRule(service string, rate float64) SamplingRule {
	return SamplingRule{
		exactService: service,
		Rate:         rate,
	}
}

// NameRule returns a SamplingRule that applies the provided sampling rate
// to spans that match the operation name provided.
func NameRule(name string, rate float64) SamplingRule {
	return SamplingRule{
		exactName: name,
		Rate:      rate,
	}
}

// NameServiceRule returns a SamplingRule that applies the provided sampling rate
// to spans matching both the operation and service names provided.
func NameServiceRule(name string, service string, rate float64) SamplingRule {
	return SamplingRule{
		exactService: service,
		exactName:    name,
		Rate:         rate,
	}
}

// RateRule returns a SamplingRule that applies the provided sampling rate to all spans.
func RateRule(rate float64) SamplingRule {
	return SamplingRule{
		Rate: rate,
	}
}

// match returns true when the span's details match all the expected values in the rule.
func (sr *SamplingRule) match(s *span) bool {
	if sr.Service != nil && !sr.Service.MatchString(s.Service) {
		return false
	} else if sr.exactService != "" && sr.exactService != s.Service {
		return false
	}
	if sr.Name != nil && !sr.Name.MatchString(s.Name) {
		return false
	} else if sr.exactName != "" && sr.exactName != s.Name {
		return false
	}
	return true
}

// rateLimiter is a wrapper on top of golang.org/x/time/rate which implements a rate limiter but also
// returns the effective rate of allowance.
type rateLimiter struct {
	limiter *rate.Limiter

	mu       sync.Mutex // guards below fields
	prevTime time.Time  // time at which prevRate was set
	prevRate float64    // previous second's rate.
	allowed  int        // number of spans allowed in the current period
	seen     int        // number of spans seen in the current period
}

// allowOne returns the rate limiter's decision to allow the span to be sampled, and the
// effective rate at the time it is called. The effective rate is computed by averaging the rate
// for the previous second with the current rate
func (r *rateLimiter) allowOne(now time.Time) (bool, float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if d := now.Sub(r.prevTime); d >= time.Second {
		// enough time has passed to reset the counters
		if d.Truncate(time.Second) == time.Second && r.seen > 0 {
			// exactly one second, so update prevRate
			r.prevRate = float64(r.allowed) / float64(r.seen)
		} else {
			// more than one second, so reset previous rate
			r.prevRate = 0.0
		}
		r.prevTime = now
		r.allowed = 0
		r.seen = 0
	}

	r.seen++
	var sampled bool
	if r.limiter.AllowN(now, 1) {
		r.allowed++
		sampled = true
	}
	// TODO(x): This algorithm is wrong. When there were no spans in the previous period prevRate will be 0.0
	// and the resulting effective rate will be half of the actual rate. We should fix the algorithm by using
	// a similar method as we do in the Datadog Agent in the rate limiter (using a decay period).
	er := (r.prevRate + (float64(r.allowed) / float64(r.seen))) / 2.0
	return sampled, er
}
