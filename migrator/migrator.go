package migrator

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/exo-framework/exo/common"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MigrateDir bool

const (
	Up   MigrateDir = true
	Down MigrateDir = false
)

// Migrator is a struct that holds the models to be migrated.
type Migrator struct {
	isInitialized      bool
	db                 *gorm.DB
	models             []any
	migrations         []string
	executedMigrations []string
}

// New creates a new migrator instance.
func New() *Migrator {
	return &Migrator{
		models:             []any{},
		migrations:         []string{},
		executedMigrations: []string{},
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

	if err := m.loadMigrationFiles(); err != nil {
		return fmt.Errorf("error loading migration files: %w", err)
	}

	if err := m.loadExecutedMigrations(); err != nil {
		return fmt.Errorf("error loading executed migrations: %w", err)
	}

	return nil
}

// AddModel adds a model to the migrator.
func (m *Migrator) AddModel(models ...any) {
	m.models = append(m.models, models...)
}

// Generates a new and empty migration file.
func (m *Migrator) GenereteEmptyMigration() (string, error) {
	return m.createMigrationFile([]string{"# Fill out as you need"}, []string{"# Fill out as you need"})
}

// Generates a diff migration file. A diff migration file is a migration file that contains the changes between the current database schema and the models.
// If asInitial is true, the diff migration file will contain the up migration code to create the database schema from scratch.
func (m *Migrator) GenerateDiffMigration(asInitial bool, gormSchemaData string) (string, error) {
	gormSchema, err := dataStringToGormSchema(gormSchemaData)
	if err != nil {
		return "", err
	}

	upSqlCode, downSqlCode, err := m.generateDiffUpAndDownCode(asInitial, gormSchema)
	if err != nil {
		return "", err
	}

	if len(upSqlCode) == 0 || len(downSqlCode) == 0 {
		return "", fmt.Errorf("no changes detected")
	}

	return m.createMigrationFile(upSqlCode, downSqlCode)
}

// ListMigrations lists all migrations.
func (m *Migrator) ListMigrations() error {
	println()
	println("Executed Migrations:")
	for _, version := range m.executedMigrations {
		print("  -", version)

		if m.isUpDeleted(version) {
			print(" (up deleted)")
		}

		if m.isDownDeleted(version) {
			print(" (down deleted)")
		}

		println()
	}

	println()
	println("Pending Migrations:")
	for _, version := range m.migrations {
		if !m.isExecuted(version) {
			println("  -", version)
		}
	}

	return nil
}

// ExecuteAll migrates all migrations which are not executed (if dir is Up) or all migrations which are executed (if dir is Down).
func (m *Migrator) ExecuteAll(dir MigrateDir) error {
	if dir == Up {
		for _, version := range m.migrations {
			if !m.isExecuted(version) {
				if err := m.Execute(version, dir); err != nil {
					return err
				}
			}
		}
	} else if dir == Down {
		a := append([]string{}, m.executedMigrations...)
		slices.Reverse(a)

		for _, version := range a {
			if err := m.Execute(version, dir); err != nil {
				return err
			}
		}
	}

	return nil
}

// Execute migrates a single migration.
func (m *Migrator) Execute(version string, dir MigrateDir) error {
	p := m.getMigrationFilePath(version, dir)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		println("WARNING: migration file not found:", p)
		return nil
	}

	if dir == Up {
		if m.isExecuted(version) {
			println("WARNING: migration already executed:", version)
			return nil
		}

		println("Uping migration:", version)
	} else if dir == Down {
		if !m.isExecuted(version) {
			println("WARNING: migration not executed:", version)
			return nil
		}

		println("Downing migration:", version)
	} else {
		return fmt.Errorf("invalid migration direction")
	}

	b, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("error reading migration file: %w", err)
	}

	code := string(b)
	code = "BEGIN;\n" + code + "\nCOMMIT;"

	tx := m.db.Begin()
	defer tx.Rollback()

	if err := tx.Exec(code).Error; err != nil {
		return fmt.Errorf("error executing migration: %w", err)
	}

	if dir == Up {
		if err := tx.Exec("INSERT INTO __migrations__ (version) VALUES (?)", version).Error; err != nil {
			return fmt.Errorf("error updating migrations table: %w", err)
		}
	} else if dir == Down {
		if err := tx.Exec("DELETE FROM __migrations__ WHERE version = ?", version).Error; err != nil {
			return fmt.Errorf("error updating migrations table: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	println("Migration executed:", version)

	return nil
}

// LoadGormSchemaForExternal loads the Gorm schema by starting the current Go code base and calling the Gorm schema callback.
func (m *Migrator) LoadExternalGormSchema() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting current working directory: %w", err)
	}

	cmd := exec.Command("go", "run", ".", "--exo-migrator-callback")
	cmd.Dir = cwd

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error running go command: %w", err)
	}

	return string(out), nil
}

func (m *Migrator) isExecuted(version string) bool {
	for _, executed := range m.executedMigrations {
		if executed == version {
			return true
		}
	}

	return false
}

func (m *Migrator) isUpDeleted(version string) bool {
	p := m.getMigrationFilePath(version, Up)
	_, err := os.Stat(p)
	return os.IsNotExist(err)
}

func (m *Migrator) isDownDeleted(version string) bool {
	p := m.getMigrationFilePath(version, Down)
	_, err := os.Stat(p)
	return os.IsNotExist(err)
}

func (m *Migrator) getMigrationFilePath(version string, dir MigrateDir) string {
	if dir == Up {
		return path.Join(getMigrationsDir(), fmt.Sprintf("%s.up.sql", version))
	}

	return path.Join(getMigrationsDir(), fmt.Sprintf("%s.down.sql", version))
}

func (m *Migrator) loadExecutedMigrations() error {
	if !m.isInitialized {
		return fmt.Errorf("migration manager is not initialized")
	}

	if err := m.db.Table("__migrations__").Select("version").Find(&m.executedMigrations).Error; err != nil {
		return fmt.Errorf("error loading executed migrations: %w", err)
	}

	return nil
}

func (m *Migrator) createMigrationFile(upCode []string, downCode []string) (string, error) {
	dir := getMigrationsDir()
	version := time.Now().Format("20060102150405")

	upFile, err := os.Create(path.Join(dir, fmt.Sprintf("%s.up.sql", version)))
	if err != nil {
		return "", fmt.Errorf("error creating migration file: %w", err)
	}

	defer upFile.Close()

	for _, line := range upCode {
		upFile.WriteString(line)
	}

	downFile, err := os.Create(path.Join(dir, fmt.Sprintf("%s.down.sql", version)))
	if err != nil {
		return "", fmt.Errorf("error creating migration file: %w", err)
	}

	defer downFile.Close()

	for _, line := range downCode {
		downFile.WriteString(line)
	}

	return version, nil
}

func (m *Migrator) loadMigrationFiles() error {
	dir := getMigrationsDir()

	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading migrations directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		version := strings.Split(name, ".")[0]

		if !strings.HasSuffix(name, ".up.sql") && !strings.HasSuffix(name, ".down.sql") {
			continue
		}

		if !slices.Contains(m.migrations, version) {
			m.migrations = append(m.migrations, version)
		}
	}

	slices.Sort(m.migrations)

	return nil
}

func getMigrationsDir() string {
	p := ".migrations"
	if _, err := os.Stat(p); os.IsNotExist(err) {
		os.Mkdir(p, os.ModePerm)
	}

	return p
}
