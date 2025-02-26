package migrator

import "strings"

func strptr(s string) *string {
	return &s
}

func normalizeDbForeignKeyConstraintAction(action *string) *string {
	if action != nil && *action == "NO ACTION" {
		return strptr("")
	}

	return action
}

func normalizeGormColumnDataRelevantInfos(column *schemaColumn) *schemaColumn {
	if column.DataType == "uuid" {
		if column.DefaultValue != nil && len(*column.DefaultValue) > 0 && !strings.HasSuffix(*column.DefaultValue, ")") {
			column.DefaultValue = strptr("'" + *column.DefaultValue + "'")
		}
	} else if column.DataType == "int" {
		column.DataType = "bigint"
	} else if column.DataType == "float" {
		column.DataType = "numeric"
	} else if strings.HasPrefix(column.DataType, "varchar") {
		if column.DefaultValue != nil && !strings.HasSuffix(*column.DefaultValue, ")") {
			column.DefaultValue = strptr("'" + *column.DefaultValue + "'")
		}
	} else if column.DataType == "time" || column.DataType == "timestamp" {
		column.DataType = "timestamp with time zone"
	} else if column.DataType == "string" {
		column.DataType = "varchar(255)"
	}

	return column
}

func normalizeDatabaseColumnDataRelevantInfos(column *schemaColumn) *schemaColumn {
	if strings.HasPrefix(column.DataType, "character varying") {
		column.DataType = "varchar" + strings.TrimPrefix(column.DataType, "character varying")

		if column.DefaultValue != nil && len(*column.DefaultValue) > 0 {
			if strings.HasSuffix(*column.DefaultValue, "::character varying") {
				column.DefaultValue = strptr(strings.TrimSuffix(*column.DefaultValue, "::character varying"))
			}
		}
	} else if column.DataType == "uuid" {
		if column.DefaultValue != nil {
			if strings.HasSuffix(*column.DefaultValue, "::uuid") {
				column.DefaultValue = strptr(strings.TrimSuffix(*column.DefaultValue, "::uuid"))
			}
		}
	}

	return column
}

func replaceExistingColumn(columns []*schemaColumn, newColumn *schemaColumn) []*schemaColumn {
	for i, col := range columns {
		if col.Name == newColumn.Name {
			columns[i] = newColumn
			return columns
		}
	}

	return append(columns, newColumn)
}

func fallbackDefaultValue(column *schemaColumn) string {
	val := *column.DefaultValue

	if column.DataType == "string" || strings.HasPrefix(column.DataType, "varchar") {
		return "'" + val + "'"
	}

	return val
}

func prepend(s string, slice []string) []string {
	return append([]string{s}, slice...)
}
