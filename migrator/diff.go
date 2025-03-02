package migrator

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (m *Migrator) generateDiffUpAndDownCode(initial bool, gormSchema []*schemaTable) ([]string, []string, error) {
	dbSchema, err := m.loadDbSchema(initial)
	if err != nil {
		return nil, nil, err
	}

	code := make([]string, 0)
	footerCode := make([]string, 0)
	downCode := make([]string, 0)
	downFooterCode := make([]string, 0)
	usedTableNames := make(map[string]bool)

	for _, table := range gormSchema {
		usedTableNames[table.Name] = true

		var dbTable *schemaTable
		for _, t := range dbSchema {
			if t.Name == table.Name {
				dbTable = t
				break
			}
		}

		if dbTable == nil { // table does not exist in database, create it
			code = append(code, generateCreateTableCode(table))
			downFooterCode = append(downFooterCode, generateDropTableCode(table.Name))

			for _, column := range table.Columns {
				for _, constraint := range column.Constraints {
					if constraint.Type == schemaConstraintTypeForeignKey && constraint.ForeignKeyDetails != nil && constraint.ForeignKeyDetails.ReferenceTable != nil && constraint.ForeignKeyDetails.ReferenceColumn != nil {
						footerCode = append(footerCode, generateForeignKeyCode(table.Name, column, constraint))
						downCode = append(downCode, generateForeignKeyDropCode(table.Name, constraint))
					}
				}
			}

			continue
		}

		usedColumnNames := make(map[string]bool)

		for _, column := range table.Columns {
			usedColumnNames[column.Name] = true

			var dbColumn *schemaColumn
			for _, col := range dbTable.Columns {
				if col.Name == column.Name {
					dbColumn = col
					break
				}
			}

			if dbColumn == nil { // column does not exist in database, create it
				code = append(code, "ALTER TABLE "+table.Name+" ADD COLUMN "+generateColumnCode(column)+";")
				downFooterCode = append(downFooterCode, "ALTER TABLE "+table.Name+" DROP COLUMN "+column.Name+" CASCADE;")

				for _, constraint := range column.Constraints {
					if constraint.Type == schemaConstraintTypeForeignKey && constraint.ForeignKeyDetails != nil && constraint.ForeignKeyDetails.ReferenceTable != nil && constraint.ForeignKeyDetails.ReferenceColumn != nil {
						footerCode = append(footerCode, generateForeignKeyCode(table.Name, column, constraint))
						downCode = append(downCode, generateForeignKeyDropCode(table.Name, constraint))
					}
				}

				continue
			}

			// column exists in database, check if it needs to be altered

			usedFkNames := make(map[string]bool)

			for _, constraint := range column.Constraints {
				if constraint.Type != schemaConstraintTypeForeignKey {
					continue
				}

				usedFkNames[constraint.Name] = true

				// there are 2 cases, 1. foreign key exists in database but reference table or column changed, 2. foreign key does not exist in database, in both cases we need to drop. but only in case 1 we need to recreate it
				var dbContraint *schemaConstraint
				for _, c := range dbColumn.Constraints {
					if c.Type == schemaConstraintTypeForeignKey && c.Name == constraint.Name {
						dbContraint = c
						break
					}
				}

				if dbContraint == nil {
					footerCode = append(footerCode, generateForeignKeyCode(table.Name, column, constraint))
					downCode = append(downCode, generateForeignKeyDropCode(table.Name, constraint))
					continue
				}

				if hasConstraintsChanged(constraint, dbContraint) {
					footerCode = append(footerCode, "ALTER TABLE "+table.Name+" DROP CONSTRAINT "+constraint.Name+";")
					downCode = append(downCode, generateForeignKeyCode(table.Name, column, constraint))

					if dbContraint.ForeignKeyDetails != nil && dbContraint.ForeignKeyDetails.ReferenceTable != nil && dbContraint.ForeignKeyDetails.ReferenceColumn != nil {
						footerCode = append(footerCode, generateForeignKeyCode(table.Name, column, constraint))
						downCode = append(downCode, generateForeignKeyDropCode(table.Name, dbContraint))
					}
				}
			}

			for _, constraint := range dbColumn.Constraints {
				if constraint.Type == schemaConstraintTypeForeignKey && !usedFkNames[constraint.Name] {
					footerCode = append(footerCode, "ALTER TABLE "+table.Name+" DROP CONSTRAINT "+constraint.Name+";")
					downCode = append(downCode, generateForeignKeyCode(table.Name, column, constraint))
				}
			}

			if column.isUnique() != dbColumn.isUnique() {
				constraintName := column.uniqueConstraintName()

				if constraintName != nil {
					if column.isUnique() {
						code = append(code, "ALTER TABLE "+table.Name+" ADD CONSTRAINT "+*constraintName+" UNIQUE ("+column.Name+");")
						downFooterCode = append(downFooterCode, "ALTER TABLE "+table.Name+" DROP CONSTRAINT "+*constraintName+";")
					} else {
						code = append(code, "ALTER TABLE "+table.Name+" DROP CONSTRAINT "+*constraintName+";")
						downFooterCode = append(downFooterCode, "ALTER TABLE "+table.Name+" ADD CONSTRAINT "+*constraintName+" UNIQUE ("+column.Name+");")
					}
				}
			}

			if column.NotNull != dbColumn.NotNull {
				if column.NotNull {
					code = append(code, "ALTER TABLE "+table.Name+" ALTER COLUMN "+column.Name+" SET NOT NULL;")
					downFooterCode = append(downFooterCode, "ALTER TABLE "+table.Name+" ALTER COLUMN "+column.Name+" DROP NOT NULL;")
				} else {
					code = append(code, "ALTER TABLE "+table.Name+" ALTER COLUMN "+column.Name+" DROP NOT NULL;")
					downFooterCode = append(downFooterCode, "ALTER TABLE "+table.Name+" ALTER COLUMN "+column.Name+" SET NOT NULL;")
				}
			}
		}

		for _, column := range dbTable.Columns {
			if _, ok := usedColumnNames[column.Name]; !ok { // column does not exist in gorm, drop it
				code = append(code, "ALTER TABLE "+table.Name+" DROP COLUMN "+column.Name+" CASCADE;")
				downFooterCode = append(downFooterCode, "ALTER TABLE "+table.Name+" ADD COLUMN "+generateColumnCode(column)+";")
			}
		}
	}

	for _, table := range dbSchema {
		if _, ok := usedTableNames[table.Name]; !ok { // table does not exist in gorm, drop it
			code = append(code, "DROP TABLE IF EXISTS "+table.Name+" CASCADE;")
			downCode = prepend(generateCreateTableCode(table), downCode)
		}
	}

	return append(code, footerCode...), append(downCode, downFooterCode...), nil
}

func generateCreateTableCode(table *schemaTable) string {
	upCode := "CREATE TABLE IF NOT EXISTS " + table.Name + " ("

	primKeys := make([]string, 0)

	for i, column := range table.Columns {
		if len(column.Name) <= 0 || len(column.DataType) <= 0 {
			jsonText, _ := json.MarshalIndent(column, "", "  ")
			fmt.Printf("WARNING: table %s -> column name or data type is empty, skipping column: %s\n", table.Name, jsonText)
			continue
		}

		upCode += generateColumnCode(column)

		if i < len(table.Columns)-1 {
			upCode += ", "
		}

		if column.isPrimaryKey() {
			primKeys = append(primKeys, column.Name)
		}
	}

	if len(primKeys) > 0 {
		upCode += ", PRIMARY KEY (" + strings.Join(primKeys, ", ") + ")"
	}

	return upCode + ");"
}

func generateDropTableCode(tableName string) string {
	return "DROP TABLE IF EXISTS " + tableName + " CASCADE;"
}

func generateColumnCode(column *schemaColumn) string {
	code := column.Name + " " + column.DataType

	if column.DefaultValue != nil {
		code += " DEFAULT " + fallbackDefaultValue(column)
	}

	if column.NotNull {
		code += " NOT NULL"
	}

	if column.isUnique() {
		code += " UNIQUE"
	}

	return code
}

func generateForeignKeyCode(tableName string, column *schemaColumn, constraint *schemaConstraint) string {
	fkDetails := constraint.ForeignKeyDetails
	alterTable := "ALTER TABLE " + tableName + " ADD CONSTRAINT " + constraint.Name + " FOREIGN KEY (" + column.Name + ") REFERENCES " + *fkDetails.ReferenceTable + " (" + *fkDetails.ReferenceColumn + ")"

	if fkDetails.OnDelete != nil && len(*fkDetails.OnDelete) > 0 {
		alterTable += " ON DELETE " + *fkDetails.OnDelete
	}

	if fkDetails.OnUpdate != nil && len(*fkDetails.OnUpdate) > 0 {
		alterTable += " ON UPDATE " + *fkDetails.OnUpdate
	}

	return alterTable + ";"
}

func generateForeignKeyDropCode(tableName string, constraint *schemaConstraint) string {
	return "ALTER TABLE " + tableName + " DROP CONSTRAINT " + constraint.Name + ";"
}

func compareStrPtr(a *string, b *string) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	return *a == *b
}

func hasConstraintsChanged(a *schemaConstraint, b *schemaConstraint) bool {
	if a.ForeignKeyDetails != nil && b.ForeignKeyDetails != nil {
		if !compareStrPtr(a.ForeignKeyDetails.ReferenceTable, b.ForeignKeyDetails.ReferenceTable) ||
			!compareStrPtr(a.ForeignKeyDetails.ReferenceColumn, b.ForeignKeyDetails.ReferenceColumn) ||
			!compareStrPtr(a.ForeignKeyDetails.OnDelete, b.ForeignKeyDetails.OnDelete) ||
			!compareStrPtr(a.ForeignKeyDetails.OnUpdate, b.ForeignKeyDetails.OnUpdate) {
			return true
		}
	}

	return false
}
