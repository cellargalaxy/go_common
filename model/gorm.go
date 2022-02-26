package model

import "time"

type Model struct {
	Id        int       `json:"id" form:"id" query:"id" gorm:"id;auto_increment;primary_key"`
	CreatedAt time.Time `json:"created_at" form:"created_at" query:"created_at" gorm:"created_at"`
	UpdatedAt time.Time `json:"updated_at" form:"updated_at" query:"updated_at" gorm:"updated_at"`
}
