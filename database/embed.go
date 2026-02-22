package database

import (
	"embed"
)

//go:embed migrations/*.sql
var MigrateSQLs embed.FS

//go:embed seeds/*/*.sql
var SeedSQLs embed.FS

// //go:embed test/*.sql
// var TestSQLs embed.FS
