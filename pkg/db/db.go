package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	p "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/source/pkger"
	"github.com/guidewire/fern-reporter/config"
	"github.com/markbates/pkger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var gdb *gorm.DB

//go:embed migrations
var migrations embed.FS

func Initialize() {
	pkger.Include("/pkg/db") //nolint //SA4017
	dbUrl := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.GetDb().Username,
		config.GetDb().Password,
		config.GetDb().Host,
		config.GetDb().Port,
		config.GetDb().Database,
	)

	pdb, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalln(err)
	}

	driver, err := p.WithInstance(pdb, &p.Config{})
	if err != nil {
		log.Fatalln(err)
	}

	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		log.Fatalln(err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		log.Fatalln(err)
	}
	if err := m.Up(); errors.Is(err, migrate.ErrNoChange) {
		log.Println(err)
	} else if err != nil {
		log.Fatalln(err)
	}

	gdb, err = gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	gdb = gdb.Debug()
}

func GetDb() *gorm.DB {
	return gdb
}

func CloseDb() {
	sqlDB, _ := gdb.DB()
	sqlDB.Close()
}
