// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build !windows

package tracer

import "time"

// now returns current UTC time in nanos.
func now() int64 {
	return time.Now().UTC().UnixNano()
}
