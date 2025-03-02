package migrator

import (
	"fmt"

	"github.com/goccy/go-json"
	"gorm.io/gorm"
)

func (m *Migrator) LoadGormSchemaForExternal() (string, error) {
	schema, err := m.loadGormSchema()
	if err != nil {
		return "", err
	}

	return gormSchemaToDataString(schema)
}

func (m *Migrator) loadGormSchema() ([]*schemaTable, error) {
	tables := make([]*schemaTable, 0)
	cachedForeignKeys := make([]*relationshipInfo, 0)

	for _, model := range m.models {
		stmt := gorm.Statement{DB: m.db}
		if err := stmt.Parse(model); err != nil {
			return nil, err
		}

		table := schemaTable{
			Name:         stmt.Schema.Table,
			Columns:      []*schemaColumn{},
			Dependencies: 0,
		}

		usedFkNames := make(map[string]int)

		for _, field := range stmt.Schema.Fields {
			if len(field.DBName) <= 0 {
				continue
			}

			column := schemaColumn{
				Name:        field.DBName,
				DataType:    string(field.DataType),
				NotNull:     field.NotNull,
				Constraints: []*schemaConstraint{},
				Indexes:     []*schemaIndex{},
			}

			if field.HasDefaultValue {
				column.DefaultValue = &field.DefaultValue
			}

			if field.Unique {
				constraint := &schemaConstraint{
					Name: stmt.Schema.Table + "_" + field.DBName + "_key",
					Type: schemaConstraintTypeUnique,
				}

				column.Constraints = append(column.Constraints, constraint)
			}

			if field.PrimaryKey {
				constraint := &schemaConstraint{
					Name: stmt.Schema.Table + "_pkey",
					Type: schemaConstraintTypePrimaryKey,
				}

				column.Constraints = append(column.Constraints, constraint)
				column.NotNull = true
			}

			table.Columns = replaceExistingColumn(table.Columns, normalizeGormColumnDataRelevantInfos(&column))
		}

		for _, rel := range stmt.Schema.Relationships.Relations {
			if rel.Type == "belongs_to" || rel.Type == "has_one" || rel.Type == "has_many" {
				relConstraint := rel.ParseConstraint()

				if relConstraint != nil {
					for _, ref := range rel.References {
						fkName := relConstraint.Name

						if _, ok := usedFkNames[fkName]; ok {
							usedFkNames[fkName]++
							fkName += "_" + fmt.Sprint(usedFkNames[fkName])
						} else {
							usedFkNames[fkName] = 1
						}

						relInfo := &relationshipInfo{
							Name:         fkName,
							FromTable:    stmt.Schema.Table,
							FromColumn:   ref.ForeignKey.DBName,
							ToTable:      relConstraint.ReferenceSchema.Table,
							ToColumn:     ref.PrimaryKey.DBName,
							RelationType: string(rel.Type),
							OnDelete:     nil,
							OnUpdate:     nil,
						}

						if relConstraint.OnDelete != "NO ACTION" {
							relInfo.OnDelete = &relConstraint.OnDelete
						}
						if relConstraint.OnUpdate != "NO ACTION" {
							relInfo.OnUpdate = &relConstraint.OnUpdate
						}

						cachedForeignKeys = append(cachedForeignKeys, relInfo)
					}
				}
			} else if rel.Type == "many_to_many" {
				joinTable := schemaTable{
					Name:    rel.JoinTable.Table,
					Columns: []*schemaColumn{},
				}

				for _, field := range rel.JoinTable.Fields {
					column := schemaColumn{
						Name:        field.DBName,
						DataType:    string(field.DataType),
						NotNull:     field.NotNull,
						Constraints: []*schemaConstraint{},
						Indexes:     []*schemaIndex{},
					}

					if len(field.DBName) <= 0 {
						continue
					}

					if field.HasDefaultValue {
						column.DefaultValue = &field.DefaultValue
					}

					if field.Unique {
						constraint := &schemaConstraint{
							Name: rel.JoinTable.Table + "_" + field.DBName + "_key",
							Type: schemaConstraintTypeUnique,
						}

						column.Constraints = append(column.Constraints, constraint)
					}

					if field.PrimaryKey {
						constraint := &schemaConstraint{
							Name: rel.JoinTable.Table + "_pkey",
							Type: schemaConstraintTypePrimaryKey,
						}

						column.Constraints = append(column.Constraints, constraint)
						column.NotNull = true
					}

					joinTable.Columns = replaceExistingColumn(joinTable.Columns, normalizeGormColumnDataRelevantInfos(&column))
				}

				for _, ref := range rel.References {
					fkName := rel.JoinTable.Table + "_" + ref.ForeignKey.DBName + "_fkey"
					relConstraint := rel.ParseConstraint()

					if _, ok := usedFkNames[fkName]; ok {
						usedFkNames[fkName]++
						fkName += "_" + fmt.Sprint(usedFkNames[fkName])
					} else {
						usedFkNames[fkName] = 1
					}

					relInfo := &relationshipInfo{
						Name:         fkName,
						FromTable:    rel.JoinTable.Table,
						FromColumn:   ref.ForeignKey.DBName,
						ToTable:      relConstraint.ReferenceSchema.Table,
						ToColumn:     ref.PrimaryKey.DBName,
						RelationType: string(rel.Type),
						OnDelete:     nil,
						OnUpdate:     nil,
					}

					if relConstraint.OnDelete != "NO ACTION" {
						relInfo.OnDelete = &relConstraint.OnDelete
					}

					if relConstraint.OnUpdate != "NO ACTION" {
						relInfo.OnUpdate = &relConstraint.OnUpdate
					}

					cachedForeignKeys = append(cachedForeignKeys, relInfo)
				}

				tables = append(tables, &joinTable)
			}
		}

		tables = append(tables, &table)
	}

	for _, rel := range cachedForeignKeys {
		fkConstraint := &schemaConstraint{
			Name: rel.Name,
			Type: schemaConstraintTypeForeignKey,
			ForeignKeyDetails: &schemaForeignKeyDetails{
				ReferenceTable:  &rel.ToTable,
				ReferenceColumn: &rel.ToColumn,
				OnDelete:        rel.OnDelete,
				OnUpdate:        rel.OnUpdate,
			},
		}

		for _, table := range tables {
			if table.Name == rel.FromTable {
				for _, col := range table.Columns {
					if col.Name == rel.FromColumn {
						col.Constraints = append(col.Constraints, fkConstraint)

						table.Dependencies++
						break
					}
				}
				break
			}
		}
	}

	return tables, nil
}

func dataStringToGormSchema(data string) ([]*schemaTable, error) {
	schema := make([]*schemaTable, 0)
	if err := json.Unmarshal([]byte(data), &schema); err != nil {
		return nil, err
	}

	return schema, nil
}

func gormSchemaToDataString(schema []*schemaTable) (string, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
