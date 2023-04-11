package main

import "github.com/auxten/postgresql-parser/pkg/sql/types"

type Column struct {
	Name    string
	Type    *types.T
	NotNull bool
	Unique  bool

	GenerationType *GenerationType
}

type UniqueConstraint = []string

type ForeignKeyRef struct {
	Table       *Table
	TableSchema string
	TableName   string
	Columns     []string
}

type ForeignKeyConstraint = struct {
	Columns []string
	Ref     *ForeignKeyRef
}

type Table struct {
	Schema                string
	Name                  string
	Columns               map[string]*Column
	PrimaryKey            []string
	UniqueConstraints     []*UniqueConstraint
	ForeignKeyConstraints []*ForeignKeyConstraint

	TableGenerationSettings *TableGenerationSettings
}

type Schema struct {
	Name   string
	Tables map[string]*Table
}
