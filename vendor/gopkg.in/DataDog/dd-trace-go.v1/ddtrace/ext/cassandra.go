// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package ext

const (
	// CassandraQuery is the tag name used for cassandra queries.
	CassandraQuery = "cassandra.query"

	// CassandraConsistencyLevel is the tag name to set for consitency level.
	CassandraConsistencyLevel = "cassandra.consistency_level"

	// CassandraCluster specifies the tag name that is used to set the cluster.
	CassandraCluster = "cassandra.cluster"

	// CassandraRowCount specifies the tag name to use when settings the row count.
	CassandraRowCount = "cassandra.row_count"

	// CassandraKeyspace is used as tag name for setting the key space.
	CassandraKeyspace = "cassandra.keyspace"

	// CassandraPaginated specifies the tag name for paginated queries.
	CassandraPaginated = "cassandra.paginated"
)
