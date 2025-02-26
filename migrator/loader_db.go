package migrator

import "slices"

var excludedTables = []string{
	"spatial_ref_sys", // if PostGis is enabled
}

func (m *Migrator) loadDbSchema(initial bool) ([]*schemaTable, error) {
	if initial {
		return []*schemaTable{}, nil
	}

	var results []detailedSchemaInfo
	err := m.db.Raw("SELECT * FROM detailed_schema_info").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	tablesMap := make(map[string]*schemaTable)
	for _, info := range results {
		if slices.Contains(excludedTables, info.TableName) {
			continue
		}

		if _, ok := tablesMap[info.TableName]; !ok {
			tablesMap[info.TableName] = &schemaTable{
				Name:    info.TableName,
				Columns: []*schemaColumn{},
			}
		}

		var column *schemaColumn
		for i, col := range tablesMap[info.TableName].Columns {
			if col.Name == info.ColumnName {
				column = tablesMap[info.TableName].Columns[i]
				break
			}
		}

		if column == nil {
			newColumn := normalizeDatabaseColumnDataRelevantInfos(&schemaColumn{
				Name:         info.ColumnName,
				DataType:     info.DataType,
				Constraints:  []*schemaConstraint{},
				Indexes:      []*schemaIndex{},
				NotNull:      info.IsNotNull,
				DefaultValue: info.DefaultValue,
			})

			tablesMap[info.TableName].Columns = append(tablesMap[info.TableName].Columns, newColumn)
			column = newColumn
		}
		if info.ConstraintName != nil {
			constraintType := schemaConstraintType(*info.ConstraintType)

			if constraintType == schemaConstraintTypeForeignKey {
				constraint := &schemaConstraint{
					Name: *info.ConstraintName,
					Type: constraintType,
					ForeignKeyDetails: &schemaForeignKeyDetails{
						ReferenceTable:  info.ForeignTableName,
						ReferenceColumn: info.ForeignColumnName,
						OnDelete:        normalizeDbForeignKeyConstraintAction(info.OnDelete),
						OnUpdate:        normalizeDbForeignKeyConstraintAction(info.OnUpdate),
					},
				}
				column.Constraints = append(column.Constraints, constraint)
			} else {
				constraint := &schemaConstraint{
					Name:              *info.ConstraintName,
					Type:              constraintType,
					ForeignKeyDetails: nil,
				}
				column.Constraints = append(column.Constraints, constraint)
			}
		}
		if info.IndexName != nil {
			index := &schemaIndex{
				Name: info.IndexName,
			}
			column.Indexes = append(column.Indexes, index)
		}
	}

	var tables []*schemaTable
	for _, table := range tablesMap {
		tables = append(tables, table)
	}

	return tables, nil
}
