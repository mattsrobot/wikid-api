package model

type SchemaMigrations struct {
	Version int64 `sql:"primary_key"`
	Dirty   bool
}
