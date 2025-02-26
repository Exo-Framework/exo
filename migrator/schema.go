package migrator

type schemaConstraintType string

const (
	schemaConstraintTypePrimaryKey schemaConstraintType = "P"
	schemaConstraintTypeUnique     schemaConstraintType = "U"
	schemaConstraintTypeForeignKey schemaConstraintType = "F"
)

type (
	schemaTable struct {
		Name         string
		Columns      []*schemaColumn
		Dependencies int
	}

	schemaColumn struct {
		Name         string
		DataType     string
		NotNull      bool
		DefaultValue *string
		Constraints  []*schemaConstraint
		Indexes      []*schemaIndex
	}

	schemaConstraint struct {
		Name              string
		Type              schemaConstraintType
		ForeignKeyDetails *schemaForeignKeyDetails
	}

	schemaForeignKeyDetails struct {
		ReferenceTable  *string
		ReferenceColumn *string
		OnDelete        *string
		OnUpdate        *string
	}

	schemaIndex struct {
		Name *string
	}

	detailedSchemaInfo struct {
		TableName         string
		ColumnName        string
		DataType          string
		ConstraintName    *string
		ConstraintType    *string
		ForeignTableName  *string
		ForeignColumnName *string
		OnDelete          *string
		OnUpdate          *string
		IndexName         *string
		IsNotNull         bool
		DefaultValue      *string
	}

	relationshipInfo struct {
		Name         string
		FromTable    string
		FromColumn   string
		ToTable      string
		ToColumn     string
		RelationType string
		OnDelete     *string
		OnUpdate     *string
	}
)

func (column *schemaColumn) isUnique() bool {
	for _, constraint := range column.Constraints {
		if constraint.Type == schemaConstraintTypeUnique {
			return true
		}
	}

	return false
}

func (column *schemaColumn) isPrimaryKey() bool {
	for _, constraint := range column.Constraints {
		if constraint.Type == schemaConstraintTypePrimaryKey {
			return true
		}
	}

	return false
}

func (column *schemaColumn) uniqueConstraintName() *string {
	for _, constraint := range column.Constraints {
		if constraint.Type == schemaConstraintTypeUnique {
			return &constraint.Name
		}
	}

	return nil
}
