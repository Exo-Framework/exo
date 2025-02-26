package migrator

type schema struct {
	Name    string
	Columns []*column
	Indexes map[string][]string
}

type column struct {
	Name       string
	Type       string
	Default    *string
	NotNull    bool
	Unique     bool
	PrimaryKey bool
	ForeignKey *foreignKey
}

type foreignKey struct {
	RefTable  string
	RefColumn string
	OnDelete  *string
	OnUpdate  *string
}
