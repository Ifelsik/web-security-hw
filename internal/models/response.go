package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Response struct {
	gorm.Model
	StatusCode int
	RequestID  uint `gorm:"uniqueIndex; not null"`
	Message    string
	Headers    datatypes.JSON
	Body       []byte
}
