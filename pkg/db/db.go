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

func Init() {
	pkger.Include("/pkg/db")
	dbUrl := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.GetDb().Username,
		config.GetDb().Password,
		config.GetDb().Host,
		config.GetDb().Port,
		config.GetDb().Database,
	)

	pdb, _ := sql.Open("postgres", dbUrl)
	driver, _ := p.WithInstance(pdb, &p.Config{})
	source, _ := iofs.New(migrations, "migrations")
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
}
