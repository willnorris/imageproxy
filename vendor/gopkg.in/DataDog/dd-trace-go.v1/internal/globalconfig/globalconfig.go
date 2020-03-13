// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// Package globalconfig stores configuration which applies globally to both the tracer
// and integrations.
package globalconfig

import (
	"math"
	"sync"
)

var cfg = &config{
	analyticsRate: math.NaN(),
}

type config struct {
	mu            sync.RWMutex
	analyticsRate float64
}

// AnalyticsRate returns the sampling rate at which events should be marked. It uses
// synchronizing mechanisms, meaning that for optimal performance it's best to read it
// once and store it.
func AnalyticsRate() float64 {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	return cfg.analyticsRate
}

// SetAnalyticsRate sets the given event sampling rate globally.
func SetAnalyticsRate(rate float64) {
	cfg.mu.Lock()
	cfg.analyticsRate = rate
	cfg.mu.Unlock()
}
