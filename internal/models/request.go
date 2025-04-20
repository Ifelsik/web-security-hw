package models

import (
	"gorm.io/gorm"
	"gorm.io/datatypes"
)

type Request struct {
	gorm.Model
	Method string
	Path string
	GetParams datatypes.JSON
	PostParams datatypes.JSON
	Headers datatypes.JSON
	Cookies datatypes.JSON
	Body string
}
