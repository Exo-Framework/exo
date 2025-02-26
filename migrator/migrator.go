package migrator

import (
	"fmt"
	"os"

	"github.com/exo-framework/exo/common"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Migrator is a struct that holds the models to be migrated.
type Migrator struct {
	isInitialized bool
	models        []interface{}
	db            *gorm.DB
}

// New creates a new migrator instance.
func New() *Migrator {
	return &Migrator{
		models: []interface{}{},
	}
}

// Initialize prepares the database for migration. If the connection fails, it returns an error.
func (m *Migrator) Initialize(edb *gorm.DB) error {
	if m.isInitialized {
		return nil
	}

	m.isInitialized = true

	hostKey := "DB_HOST"
	portKey := "DB_PORT"
	userKey := "DB_USER"
	passKey := "DB_PASS"
	nameKey := "DB_NAME"

	if edb != nil {
		m.db = edb
	} else {
		rc := common.LoadRuntimeConfig()
		if oHostKey, ok := rc["DB_HOST"]; ok {
			hostKey = oHostKey
		}

		if oPortKey, ok := rc["DB_PORT"]; ok {
			portKey = oPortKey
		}

		if oUserKey, ok := rc["DB_USER"]; ok {
			userKey = oUserKey
		}

		if oPassKey, ok := rc["DB_PASS"]; ok {
			passKey = oPassKey
		}

		if oNameKey, ok := rc["DB_NAME"]; ok {
			nameKey = oNameKey
		}

		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s", os.Getenv(hostKey), os.Getenv(portKey), os.Getenv(userKey), os.Getenv(passKey), os.Getenv(nameKey))

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return err
		}

		m.db = db
	}

	if err := m.db.Exec("CREATE TABLE IF NOT EXISTS __migrations__ (version VARCHAR(255) PRIMARY KEY)").Error; err != nil {
		return fmt.Errorf("error creating migrations table: %w", err)
	}

	if err := m.db.Exec("CREATE OR REPLACE VIEW detailed_schema_info AS SELECT tbl.relname AS table_name, att.attname AS column_name, CASE WHEN isc.character_maximum_length IS NOT NULL THEN att.atttypid::regtype::text || '(' || isc.character_maximum_length::text || ')' ELSE att.atttypid::regtype::text END AS data_type, idx.relname AS index_name, con.conname AS constraint_name, CASE WHEN con.contype = 'p' THEN 'PRIMARY KEY' WHEN con.contype = 'f' THEN 'FOREIGN KEY' WHEN con.contype = 'u' THEN 'UNIQUE' ELSE con.contype END AS constraint_type, fk_info.foreign_table_name, fk_info.foreign_column_name, CASE fk_info.confdeltype WHEN 'a' THEN 'NO ACTION' WHEN 'r' THEN 'RESTRICT' WHEN 'c' THEN 'CASCADE' WHEN 'n' THEN 'SET NULL' WHEN 'd' THEN 'SET DEFAULT' END AS on_delete, CASE fk_info.confupdtype WHEN 'a' THEN 'NO ACTION' WHEN 'r' THEN 'RESTRICT' WHEN 'c' THEN 'CASCADE' WHEN 'n' THEN 'SET NULL' WHEN 'd' THEN 'SET DEFAULT' END AS on_update, isc.column_default AS default_value, isc.is_nullable = 'NO' AS is_not_null FROM pg_attribute att JOIN pg_class tbl ON att.attrelid = tbl.oid JOIN pg_namespace nsp ON tbl.relnamespace = nsp.oid LEFT JOIN pg_index ind ON att.attrelid = ind.indrelid AND att.attnum = ANY(ind.indkey) LEFT JOIN pg_class idx ON ind.indexrelid = idx.oid LEFT JOIN pg_constraint con ON att.attrelid = con.conrelid AND att.attnum = ANY(con.conkey) LEFT JOIN information_schema.columns isc ON isc.table_name = tbl.relname AND isc.column_name = att.attname LEFT JOIN ( SELECT con.oid, con.conrelid, con.conkey, clf.relname AS foreign_table_name, af.attname AS foreign_column_name, confdeltype, confupdtype FROM pg_constraint con JOIN pg_class clf ON con.confrelid = clf.oid JOIN pg_namespace nf ON clf.relnamespace = nf.oid JOIN pg_attribute af ON af.attrelid = clf.oid AND af.attnum = ANY(con.confkey) WHERE con.contype = 'f' ) AS fk_info ON con.oid = fk_info.oid WHERE nsp.nspname = 'public' AND tbl.relname != '__migrations__' AND tbl.relkind = 'r' AND att.attnum > 0 AND NOT att.attisdropped;").Error; err != nil {
		return fmt.Errorf("error creating detailed_schema_info view: %w", err)
	}

	return nil
}

// AddModel adds a model to the migrator.
func (m *Migrator) AddModel(model interface{}) {
	m.models = append(m.models, model)
}
