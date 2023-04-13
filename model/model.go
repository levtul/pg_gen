package model

import (
	"github.com/auxten/postgresql-parser/pkg/sql/types"
	"github.com/google/uuid"
	"github.com/lib/pq/oid"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type Column struct {
	Name    string
	Type    *types.T
	NotNull bool
	Unique  bool

	GenerationType GenerationType
}

func (c Column) GenerateValue() interface{} {
	if c.GenerationType == nil {
		switch c.Type.Family() {
		case types.IntFamily:
			return rand.Int31()
		case types.StringFamily:
			if c.Type.Oid() == oid.T_text {
				return RandStringRunes(rand.Intn(20))
			} else {
				return RandStringRunes(1)
			}
		case types.BoolFamily:
			return rand.Intn(2) == 0
		case types.FloatFamily, types.DecimalFamily:
			return rand.Float64()
		case types.DateFamily:
			return time.Now().AddDate(0, 0, rand.Intn(1000)-500)
		case types.TimestampFamily, types.TimestampTZFamily:
			return time.Now().Add(time.Duration(rand.Intn(1000)-500) * time.Hour)
		case types.TimeFamily, types.TimeTZFamily:
			return time.Now().Add(time.Duration(rand.Intn(1000)-500) * time.Second)
		case types.IntervalFamily:
			return time.Duration(rand.Intn(1000)-500) * time.Second
		case types.JsonFamily:
			return "{}"
		case types.UuidFamily:
			return uuid.New()
		default:
			return nil
		}
	} else {
		return c.GenerationType.GenerateValue()
	}
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
