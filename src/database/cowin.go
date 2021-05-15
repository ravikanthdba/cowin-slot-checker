package database

import (
	"fmt"
	"log"
	"time"

	_ "github.com/jinzhu/gorm/dialects/mysql"

	"github.com/jinzhu/gorm"
	bulk "github.com/t-tiger/gorm-bulk-insert"
)

type DatabaseConnection struct {
	Connection *gorm.DB
}

func CreateConnection(username, password, hostname, database string) (*DatabaseConnection, error) {
	db, err := gorm.Open("mysql", username+":"+password+"@tcp("+hostname+":3306)/"+database)
	if err != nil {
		return nil, fmt.Errorf("ERROR: Connecting to database: %q", err)
	}
	var conn DatabaseConnection
	conn.Connection = db
	return &conn, nil
}

func (d *DatabaseConnection) Load(data []interface{}, loadValue int) (bool, error) {
	tx := d.Connection.Begin()
	start := time.Now()
	err := bulk.BulkInsert(tx, data, loadValue)
	if err != nil {
		return false, fmt.Errorf("ERROR: Error in Bulk Loading: %q", err)
	}
	tx.Commit()
	log.Println("Bulk Load completed in: ", time.Since(start))
	return true, nil
}

func (d *DatabaseConnection) DatabaseQueryWithoutCondition(query string) (*gorm.DB, error) {
	tx := d.Connection.Begin()
	start := time.Now()
	results := d.Connection.Raw(query)
	tx.Commit()
	log.Println("Select Query completed in: ", time.Since(start))
	return results, nil
}

func (d *DatabaseConnection) DatabaseQueryWithCondition(query string, model interface{}, condition ...interface{}) (*gorm.DB, error) {
	tx := d.Connection.Begin()
	start := time.Now()
	results := d.Connection.Raw(query, condition...).Scan(model)
	tx.Commit()
	log.Println("Select Query with condition completed in: ", time.Since(start))
	return results, nil
}

func (d *DatabaseConnection) DeleteRecords(query string, model interface{}, condition ...interface{}) (*gorm.DB, error) {
	tx := d.Connection.Begin()
	start := time.Now()
	results := d.Connection.Where(query, condition...).Delete(model)
	tx.Commit()
	log.Println("Delete Query with condition for delete completed in: ", time.Since(start))
	return results, nil
}

func (d *DatabaseConnection) AutoMigrateTables(table ...interface{}) (bool, error) {
	tx := d.Connection.Begin()
	start := time.Now()
	result := tx.AutoMigrate(table...)
	if result.Error != nil {
		return false, result.Error
	}
	tx.Commit()
	log.Println("Tables have been created")
	log.Println("Tables creation completed in: ", time.Since(start))
	return true, nil
}
