// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build go1.11

package tracer

import (
	"context"
	t "runtime/trace"
)

func startExecutionTracerTask(name string) func() {
	if !t.IsEnabled() {
		return func() {}
	}
	_, task := t.NewTask(context.TODO(), name)
	return task.End
}
