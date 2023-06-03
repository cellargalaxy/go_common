package model

import "time"

type Model struct {
	Id        int       `json:"id" form:"id" query:"id" gorm:"id;auto_increment;primary_key"`
	CreatedAt time.Time `json:"created_at" form:"created_at" query:"created_at" gorm:"created_at"`
	UpdatedAt time.Time `json:"updated_at" form:"updated_at" query:"updated_at" gorm:"updated_at"`
}

type Inquiry struct {
	PageNum  int `json:"page_num" form:"page_num" query:"page_num"`
	PageSize int `json:"page_size" form:"page_size" query:"page_size"`
}

func (this Inquiry) GetPageNum() int {
	return this.PageNum
}
func (this Inquiry) GetPageSize() int {
	return this.PageSize
}
