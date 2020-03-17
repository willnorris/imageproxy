// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package ext

// Priority is a hint given to the backend so that it knows which traces to reject or kept.
// In a distributed context, it should be set before any context propagation (fork, RPC calls) to be effective.

const (
	// PriorityUserReject informs the backend that a trace should be rejected and not stored.
	// This should be used by user code overriding default priority.
	PriorityUserReject = -1

	// PriorityAutoReject informs the backend that a trace should be rejected and not stored.
	// This is used by the builtin sampler.
	PriorityAutoReject = 0

	// PriorityAutoKeep informs the backend that a trace should be kept and not stored.
	// This is used by the builtin sampler.
	PriorityAutoKeep = 1

	// PriorityUserKeep informs the backend that a trace should be kept and not stored.
	// This should be used by user code overriding default priority.
	PriorityUserKeep = 2
)
