package main

import (
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/types"
	"github.com/go-faker/faker/v4"
	"math/rand"
	"reflect"
	"strings"
)

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

func String(p GenerationPreset) string {
	switch p {
	case GenerationPresetName:
		return "name"
	case GenerationPresetSurname:
		return "surname"
	case GenerationPresetPatronymic:
		return "patronymic"
	case GenerationPresetNameRu:
		return "name_ru"
	case GenerationPresetSurnameRu:
		return "surname_ru"
	case GenerationPresetPatronymicRu:
		return "patronymic_ru"
	case GenerationPresetAddress:
		return "address"
	case GenerationPresetAddressRu:
		return "address_ru"
	case GenerationPresetPhone:
		return "phone"
	case GenerationPresetEmail:
		return "email"
	}

	return ""
}

func GenerationPresetFromString(s string) (GenerationPreset, error) {
	switch s {
	case "name":
		return GenerationPresetName, nil
	case "surname":
		return GenerationPresetSurname, nil
	case "patronymic":
		return GenerationPresetPatronymic, nil
	case "name_ru":
		return GenerationPresetNameRu, nil
	case "surname_ru":
		return GenerationPresetSurnameRu, nil
	case "patronymic_ru":
		return GenerationPresetPatronymicRu, nil
	case "address":
		return GenerationPresetAddress, nil
	case "address_ru":
		return GenerationPresetAddressRu, nil
	case "phone":
		return GenerationPresetPhone, nil
	case "email":
		return GenerationPresetEmail, nil
	}

	return 0, fmt.Errorf("unknown generation preset: %s", s)
}

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
	CommentString() string
	ValidateType(t *types.T) error
	SetValue(v string) error
	GenerateValue() interface{}
}

func (*GenerationTypeOneof) generationType()  {}
func (*GenerationTypeRange) generationType()  {}
func (*GenerationTypePreset) generationType() {}

func (*GenerationTypeOneof) CommentString() string {
	return "oneof"
}
func (*GenerationTypeRange) CommentString() string {
	return "range"
}
func (*GenerationTypePreset) CommentString() string {
	return "type"
}

func (gto *GenerationTypeOneof) SetValue(v string) error {
	if len(v) == 0 {
		return fmt.Errorf("invalid oneof value: %s", v)
	} else if v[0] != '[' || v[len(v)-1] != ']' {
		return fmt.Errorf("invalid oneof value: %s", v)
	} else {
		v = v[1 : len(v)-1]
	}

	arr := strings.Split(v, ",")
	if len(arr) == 0 {
		return fmt.Errorf("invalid oneof value: %s", v)
	}

	gto.Values = make([]interface{}, len(arr))
	for i, v := range arr {
		gto.Values[i] = v
	}
	return nil
}
func (gtr *GenerationTypeRange) SetValue(v string) error {
	if len(v) == 0 {
		return fmt.Errorf("invalid range value: %s", v)
	} else if v[0] != '[' || v[len(v)-1] != ']' {
		return fmt.Errorf("invalid range value: %s", v)
	} else {
		v = v[1 : len(v)-1]
	}
	arr := strings.Split(v, " - ")
	if len(arr) != 2 || arr[0] == "" || arr[1] == "" {
		return fmt.Errorf("invalid range value: %s", v)
	}

	gtr.From = arr[0]
	gtr.To = arr[1]
	return nil
}
func (gtp *GenerationTypePreset) SetValue(v string) error {
	p, err := GenerationPresetFromString(v)
	if err != nil {
		return err
	}

	gtp.Preset = p
	return nil
}

func (gto *GenerationTypeOneof) ValidateType(t *types.T) error {
	switch t.Family() {
	case types.IntFamily, types.FloatFamily, types.DecimalFamily, types.StringFamily, types.DateFamily, types.TimestampFamily, types.TimeFamily:
		return nil
	default:
		return fmt.Errorf("generation type oneof can be used only with numeric, string, date and time types, got %s", t.String())
	}
}
func (gtr *GenerationTypeRange) ValidateType(t *types.T) error {
	switch t.Family() {
	case types.IntFamily, types.FloatFamily, types.DecimalFamily, types.DateFamily, types.TimestampFamily, types.TimeFamily:
		return nil
	default:
		return fmt.Errorf("generation type range can be used only with numeric, date and time types, got %s", t.String())
	}
}
func (gtp *GenerationTypePreset) ValidateType(t *types.T) error {
	switch gtp.Preset {
	case GenerationPresetName, GenerationPresetSurname, GenerationPresetPatronymic, GenerationPresetNameRu, GenerationPresetSurnameRu, GenerationPresetPatronymicRu, GenerationPresetEmail, GenerationPresetAddress, GenerationPresetAddressRu, GenerationPresetPhone:
		if t.Family() != types.StringFamily {
			return fmt.Errorf("generation type %s can be used only with string type, got %s", String(gtp.Preset), t.String())
		}
	default:
		return fmt.Errorf("unknown generation preset: %s", String(gtp.Preset))
	}

	return nil
}

func (gto *GenerationTypeOneof) GenerateValue() interface{} {
	return gto.Values[rand.Intn(len(gto.Values))]
}

func (gtr *GenerationTypeRange) GenerateValue() interface{} {
	return nil
}

func (gtp *GenerationTypePreset) GenerateValue() interface{} {
	switch gtp.Preset {
	case GenerationPresetName:
		return faker.FirstName()
	case GenerationPresetSurname:
		return faker.LastName()
	case GenerationPresetPatronymic:
		return faker.FirstNameMale()
	case GenerationPresetNameRu:
		s, _ := faker.GetPerson().RussianFirstNameMale(reflect.Value{})
		return s
	case GenerationPresetSurnameRu:
		s, _ := faker.GetPerson().RussianLastNameMale(reflect.Value{})
		return s
	case GenerationPresetPatronymicRu:
		s, _ := faker.GetPerson().RussianFirstNameMale(reflect.Value{})
		return s
	case GenerationPresetAddress:
		addr, _ := faker.GetAddress().RealWorld(reflect.Value{})
		a := addr.(faker.RealAddress)
		return a.Address
	case GenerationPresetAddressRu:
		addr, _ := faker.GetAddress().RealWorld(reflect.Value{})
		a := addr.(faker.RealAddress)
		return a.Address
	case GenerationPresetPhone:
		return faker.Phonenumber()
	case GenerationPresetEmail:
		return faker.Email()
	}

	return nil
}

func generationTypeFromString(s string) (res GenerationType, err error) {
	switch s {
	case "oneof":
		res = &GenerationTypeOneof{}
	case "range":
		res = &GenerationTypeRange{}
	case "type":
		res = &GenerationTypePreset{}
	default:
		err = fmt.Errorf("unknown generation type: %s", s)
	}

	return
}

func NewGenerationTypeFromString(s string, t *types.T) (*GenerationType, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("unknown generation type: %s", s)
	}

	gType := parts[0]
	val := parts[1]
	gt, err := generationTypeFromString(gType)
	if err != nil {
		return nil, err
	}

	err = gt.SetValue(val)
	if err != nil {
		return nil, err
	}

	err = gt.ValidateType(t)
	if err != nil {
		return nil, err
	}

	return &gt, nil
}

type TableGenerationSettings struct {
	RowsCount int
}
