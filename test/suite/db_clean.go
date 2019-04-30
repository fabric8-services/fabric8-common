package suite

import (
	"database/sql"
	"fmt"

	"github.com/fabric8-services/fabric8-common/log"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// DeleteCreatedEntities records all created entities on the gorm.DB connection
// and returns a function which can be called on defer to delete created
// entities in reverse order on function exit.
//
// In addition to that, the WIT cache is cleared as well in order to respect any
// deletions made to the db.
//
// Usage:
//
// func TestDatabaseActions(t *testing.T) {
//
// 	// setup database connection
// 	db := ....
// 	// setup auto clean up of created entities
// 	defer DeleteCreatedEntities(db)()
//
// 	repo := NewRepo(db)
// 	repo.Create(X)
// 	repo.Create(X)
// 	repo.Create(X)
// }
//
// Output:
//
// 2017/01/31 12:08:08 Deleting from x 6d143405-1232-40de-bc73-835b543cd972
// 2017/01/31 12:08:08 Deleting from x 0685068d-4934-4d9a-bac2-91eebbca9575
// 2017/01/31 12:08:08 Deleting from x 2d20944e-7952-40c1-bd15-f3fa1a70026d
func DeleteCreatedEntities(db *gorm.DB, config DBTestSuiteConfiguration) func() error {
	type entity struct {
		table string
		keys  map[string]interface{}
	}
	var entities []entity
	hookName := newHookname()
	db.Callback().Create().After("gorm:create").Register(hookName, func(scope *gorm.Scope) {
		fields := scope.PrimaryFields()
		keys := make(map[string]interface{})
		for _, field := range fields {
			keys[field.DBName] = field.Field.Interface()
		}
		log.Logger().Debugln(fmt.Sprintf("Inserted entities from %s with keys %v", scope.TableName(), keys))
		entities = append(entities, entity{table: scope.TableName(), keys: keys})
	})
	return func() error {
		var resultErr error
		// Find out if the current db object is already a transaction
		_, inTransaction := db.CommonDB().(*sql.Tx)
		tx := db
		if !inTransaction {
			tx = db.Begin()
		}
		defer func() {
			db.Callback().Create().Remove(hookName)
		}()
		// defer all DB constraints that can be deferred (see https://www.postgresql.org/docs/9.6/sql-set-constraints.html)
		_, err := tx.CommonDB().Exec("SET CONSTRAINTS ALL DEFERRED")
		if err != nil {
			log.Error(nil, map[string]interface{}{
				"error": err,
			}, "failed to defer all constraints before cleaning the test records in the DB")
		}
		for i := len(entities) - 1; i >= 0; i-- {
			entity := entities[i]
			log.Debug(nil, map[string]interface{}{
				"table":     entity.table,
				"keys":      entity.keys,
				"hook_name": hookName,
			}, "Deleting entities from '%s' table with keys %v", entity.table, entity.keys)
			if len(entity.keys) == 0 {
				if config.IsCleanTestDataErrorReportingRequired() {
					resultErr = fmt.Errorf("no primary keys found for '%s' table", entity.table)
					log.Error(nil, map[string]interface{}{
						"entities": entity.table,
					}, "no entity to delete?")
					break
				}
			} else {
				err := tx.Table(entity.table).Where(entity.keys).Delete("").Error
				if err != nil && config.IsCleanTestDataErrorReportingRequired() {
					log.Error(nil, map[string]interface{}{
						"error":    err,
						"entities": entity.table,
					}, "failed to delete entities")
					resultErr = errors.Wrap(err, fmt.Sprintf("failed to delete entities for '%s' table", entity.table))
					break
				}
			}
		}

		if resultErr == nil && !inTransaction {
			err := tx.Commit().Error
			if err != nil && config.IsCleanTestDataErrorReportingRequired() {
				// err = errors.Wrap(resultErr, "unable to cleanup DB")
				resultErr = errors.Wrap(err, "failed to commit transaction")
			}
		}
		return resultErr
	}
}

func newHookname() string {
	return fmt.Sprintf("fabric8:record-%s", uuid.NewV4())
}
