package models

type ModelOfUser struct {
	Id    int      `gorm:"id"`
	Roles []string `gorm:"roles"`
}
