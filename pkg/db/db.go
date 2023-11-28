package db

import (
	"database/sql"
	"embed"
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	p "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/source/pkger"
	"github.com/markbates/pkger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var gdb *gorm.DB

//go:embed migrations
var migrations embed.FS

func Init() {
	pkger.Include("/pkg/db")
	pdb, _ := sql.Open("postgres", "postgres://fern:fern@localhost:5432/fern?sslmode=disable")
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
	// pkger.Include("/pkg/db/migrations")
	// m, err := migrate.New(
	// 	"pkger://pkg/db/migrations",
	// 	"postgres://fern:fern@localhost:5432/fern?sslmode=disable")
	//
	// // Run migrations from packed files
	// if err = m.Up(); err != nil {
	// 	log.Fatal(err)
	// }
	// dbConfig := config.GetDb()
	// dbConfig := config.dbConfig{
	// 	Username: "fern",
	// 	Password: "fern",
	// 	Host:     "localhost",
	// 	Port:     "5432",
	// 	Database: "fern",
	// 	Driver:   "postgres",
	// }
	// dbinfo := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
	// 	dbConfig.Username,
	// 	dbConfig.Password,
	// 	dbConfig.Host,
	// 	dbConfig.Port,
	// 	dbConfig.Database,
	// )

	gdb, err = gorm.Open(postgres.Open("postgres://fern:fern@localhost:5432/fern?sslmode=disable"), &gorm.Config{})
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	gdb = gdb.Debug()

	// db.LogMode(dbConfig.DetailLog)
	// db.DB().SetMaxOpenConns(dbConfig.MaxOpenConns)
	// db.DB().SetMaxIdleConns(dbConfig.MaxIdleConns)
	// db.AutoMigrate(&models.TestRun{})
	// defer gdb.Close()

}

func GetDb() *gorm.DB {
	return gdb
}

func CloseDb() {
}
