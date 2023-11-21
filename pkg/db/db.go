package db

import (
	"fern-reporter/config"
	"fmt"
	"log"

	"gorm.io/gorm"
)

var db *gorm.DB

func Init() {
	dbConfig := config.GetDb()
	dbinfo := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		dbConfig.Username,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Database,
	)

	var err error
	db, err = gorm.Open(dbConfig.Driver, dbinfo)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	db.LogMode(dbConfig.DetailLog)
	db.DB().SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.DB().SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.AutoMigrate(&models.Person{})
	defer db.Close()
}

func GetDb() *gorm.DB {
	return db
}

func CloseDb() {
	db.Close()
}
