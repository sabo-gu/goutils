package db

import (
	"bytes"
	"encoding/json"
	"log"
	"time"

	"github.com/DoOR-Team/gorm"
	"github.com/rs/xid"
)

type AttributesMode struct {
	Attributes   string `gorm:"type:varchar(2048)"`
	Attribute_cc int
}

func (this *AttributesMode) SetAttribute(k, v string) {
	var buf bytes.Buffer
	m := make(map[string]string)
	json.NewDecoder(bytes.NewBufferString(this.Attributes)).Decode(&m)
	m[k] = v
	json.NewEncoder(&buf).Encode(m)
	this.Attributes = buf.String()
}

func (this AttributesMode) GetAttribute(k string) string {
	m := make(map[string]string)
	json.NewDecoder(bytes.NewBufferString(this.Attributes)).Decode(&m)
	v := m[k]
	return v
}

func (this AttributesMode) GetAttributeMap() map[string]string {
	m := make(map[string]string)
	json.NewDecoder(bytes.NewBufferString(this.Attributes)).Decode(&m)
	return m
}

type Model struct {
	ID        string `gorm:"primary_key;type:char(20)"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

type BasicModel struct {
	Model          //包括ID，CreateAt，UpdateAt，DeletedAt
	AttributesMode //包括attributes和 attribute_cc
}

type BasicUintModel struct {
	ID        string `gorm:"primary_key;type:char(20)"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
	AttributesMode
}

type BasicModelV3 struct {
	ID        string     `gorm:"primary_key;type:char(20)"`
	CreatedAt uint64     `gorm:"type:timestamp(6)"`
	UpdatedAt uint64     `gorm:"type:timestamp(6)"`
	DeletedAt *time.Time `gorm:"type:timestamp(6)";sql:"index"`
	AttributesMode
}

//使用方法如下
type UsageModel struct {
	BasicModel
	//自定义的列
	//....
}

func (this *BasicUintModel) BeforeCreate(scope *gorm.Scope) error {
	if this.ID != "" {
		return nil
	}
	xId := xid.New()
	err := scope.SetColumn("ID", xId.String())
	this.ID = xId.String()
	return err
}

func (m *BasicUintModel) BeforeBatchCreate(scope *gorm.Scope) error {
	// 找到每行的ID，拿出为blank的那些，最终批量赋予snowflake id
	indirectScopeValue := scope.IndirectValue()
	blankIDFileds := []*gorm.Field{}

	for elementIndex := 0; elementIndex < indirectScopeValue.Len(); elementIndex++ {
		fields := gorm.FiledsWithIndexForBatch(scope, elementIndex)
		for _, field := range fields {
			if field.Name != "ID" {
				continue
			}
			if field.IsBlank {
				blankIDFileds = append(blankIDFileds, field)
			}
		}
	}

	if len(blankIDFileds) <= 0 {
		return nil
	}

	ids := make([]xid.ID, len(blankIDFileds))
	for i := 0; i < len(blankIDFileds); i++ {
		ids[i] = xid.New()
	}

	for index, field := range blankIDFileds {
		err := field.Set(ids[index].String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *BasicModel) BeforeCreate(scope *gorm.Scope) error {
	if this.ID != "" {
		return nil
	}
	xId := xid.New()
	err := scope.SetColumn("ID", xId.String())
	this.ID = xId.String()
	return err
}
func (this *BasicUintModel) CreateUintID() error {
	if this.ID != "" {
		log.Println("ID(", this.ID, ") 不为0,不产生新的ID.")
		return nil
	}
	xId := xid.New()
	this.ID = xId.String()
	return nil
}
