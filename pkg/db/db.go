package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/guidewire/fern-reporter/pkg/utils"

	"github.com/golang-migrate/migrate/v4"
	p "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/source/pkger"
	"github.com/guidewire/fern-reporter/config"
	"github.com/markbates/pkger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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
		utils.GetLogger().Fatal("[ERROR]: Unable to connect to the database: ", err)
	}

	driver, err := p.WithInstance(pdb, &p.Config{})
	if err != nil {
		utils.GetLogger().Fatal("[ERROR]: Unable to create migration driver: ", err)
	}

	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		utils.GetLogger().Fatal("[ERROR]: Unable to create migration source: ", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		utils.GetLogger().Fatal("[ERROR]: Unable to create migration instance: ", err)
	}
	if err := m.Up(); errors.Is(err, migrate.ErrNoChange) {
		utils.GetLogger().Warn("[LOG]: No new migrations to apply")
	} else if err != nil {
		utils.GetLogger().Fatal("[ERROR]: Unable to run database migrations: ", err)
	}

	gdb, err = gorm.Open(postgres.Open(dbUrl), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		utils.GetLogger().Fatal("[ERROR]: Unable to connect to the database: ", err)
	}

	// gdb = gdb.Debug()
}

func GetDb() *gorm.DB {
	return gdb
}

func CloseDb() {
	sqlDB, _ := gdb.DB()
	err := sqlDB.Close()
	if err != nil {
		utils.GetLogger().Fatal("[ERROR]: Unable to close the db connection: ", err)
	}
}
