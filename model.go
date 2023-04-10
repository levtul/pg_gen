package main

import "github.com/auxten/postgresql-parser/pkg/sql/types"

type GenerationPreset = int

const (
	GenerationPresetName GenerationPreset = iota
	GenerationPresetSurname
	GenerationPresetPatronymic
	GenerationPresetNameRu
	GenerationPresetSurnameRu
	GenerationPresetPatronymicRu
	GenerationPresetAddress
	GenerationPresetAddressRu
	GenerationPresetPhone
	GenerationPresetEmail
)

type GenerationTypeOneof struct {
	Values []interface{}
}

type GenerationTypeRange struct {
	From interface{}
	To   interface{}
}

type GenerationTypePreset struct {
	Preset GenerationPreset
}

type GenerationType interface {
	generationType()
}

func (GenerationTypeOneof) generationType()  {}
func (GenerationTypeRange) generationType()  {}
func (GenerationTypePreset) generationType() {}

type TableGenerationSettings struct {
	RowsCount int
}

type Column struct {
	Name    string
	Type    *types.T
	NotNull bool

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
