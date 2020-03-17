// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package version

// Tag specifies the current release tag. It needs to be manually
// updated. A test checks that the value of Tag never points to a
// git tag that is older than HEAD.
const Tag = "v1.22.0"
