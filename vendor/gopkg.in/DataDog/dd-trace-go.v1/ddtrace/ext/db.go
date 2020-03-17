// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package ext

const (
	// DBApplication indicates the application using the database.
	DBApplication = "db.application"
	// DBName indicates the database name.
	DBName = "db.name"
	// DBType indicates the type of Database.
	DBType = "db.type"
	// DBInstance indicates the instance name of Database.
	DBInstance = "db.instance"
	// DBUser indicates the user name of Database, e.g. "readonly_user" or "reporting_user".
	DBUser = "db.user"
	// DBStatement records a database statement for the given database type.
	DBStatement = "db.statement"
)
