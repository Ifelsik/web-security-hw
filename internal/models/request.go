package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Request struct {
	gorm.Model
	Method     string
	Path       string
	TLS        bool
	GetParams  datatypes.JSON
	PostParams datatypes.JSON
	Headers    datatypes.JSON
	Cookies    datatypes.JSON
	Body       []byte
	Response   *Response `gorm:"foreignKey:RequestID"`
}
