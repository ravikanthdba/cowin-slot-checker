package database

import (
	"fmt"
	"log"
	"strconv"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	bulk "github.com/t-tiger/gorm-bulk-insert"
	// model "github.com/cowin-slot-checker/src/model"
)

type DatabaseConnection struct {
	Connection *gorm.DB
}

func Connection(username, password, hostname, database string) (*DatabaseConnection, error) {
	db, err := gorm.Open("mysql", username + ":" + password + "@tcp(" + hostname + ":3306)/" + database)
	if err != nil {
		return nil, fmt.Errorf("ERROR: Connecting to database: ", err)
	}
	var conn DatabaseConnection
	conn.Connection = db
	return &conn, nil
}

func (d *DatabaseConnection) Load(data []interface{}, loadValue int) (bool, error) {
	tx := d.Connection.Begin()
	err := bulk.BulkInsert(d.Connection, data, loadValue)
	if err != nil {
		return false, fmt.Errorf("Error in Bulk Loading: ", err)
	}
	tx.Commit()
	return true, nil
}

func (d *DatabaseConnection) DeleteRecords(model interface{}) (bool, error) {
	db := d.Connection.Unscoped().Delete(&model)
	if db.Error != nil {
		return false, fmt.Errorf("ERROR: Delete failed: ", db.Error)
	}
	log.Println(strconv.Itoa(int(db.RowsAffected)) + " records deleted from the database")
	return true, nil
}

type Hospitals struct {
	Name              string
	StateName         string
	DistrictName      string
	BlockName         string
	Pincode           int
	Lat               int
	Long              int
	Date              string
	AvailableCapacity float64
	MinAgeLimit       int
	Vaccine           string
}

func (d *DatabaseConnection) SelectRecords(condition string, model interface{}, conditionValue ...interface{}) (interface{}, error) {
	tx := d.Connection.Begin()
	hospitals := model.(Hospitals)
	results := d.Connection.Where("name = ?", conditionValue...).Find(&hospitals)
	if tx.Error != nil {
		return nil, fmt.Errorf("ERROR: Query failed: ", tx.Error)
	}
	tx.Commit()
	log.Println(results.RowsAffected)
	log.Println(results.Error)
	return results, nil
}