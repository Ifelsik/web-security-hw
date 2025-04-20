package models

import (
	"gorm.io/gorm"
	"gorm.io/datatypes"
)

type Response struct {
	gorm.Model
	StatusCode int
	Message string
	Headers datatypes.JSON
	Body string
}
