package main

import (
	"time"
    "gorm.io/gorm"
)

type User struct {
	ID string `gorm:"primaryKey" json:"id"`
	ChannelUser string `json:"channel"`
}

type Task struct {
	TaskID uint `gorm:"primaryKey;autoIncrement:false" json:"task_id"`
	Name string `gorm:"primaryKey" json:"name"`
	Description string `json:"description"`
    Date time.Time `json:"date"`
	ChannelTask string `json:"channel"`
}

type Channel struct {
	Name string `gorm:"primaryKey" json:"name"`
	Key string `json:"key"`
	Tasks []Task `gorm:"foreignKey:ChannelTask"`
	Users []User `gorm:"foreignKey:ChannelUser"`
}


func doMigration(db *gorm.DB) {
	db.AutoMigrate(&Task{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Channel{})
}