package gen

type Method string

const (
	MethodGet     Method = "Get"
	MethodPost    Method = "Post"
	MethodPut     Method = "Put"
	MethodDelete  Method = "Delete"
	MethodPatch   Method = "Patch"
	MethodOptions Method = "Options"
	MethodHead    Method = "Head"
	MethodTrace   Method = "Trace"
)

type FieldType string

const (
	FieldPath   FieldType = "path"
	FieldQuery  FieldType = "query"
	FieldHeader FieldType = "header"
	FieldBody   FieldType = "body"
	FieldForm   FieldType = "form"
)

type Field struct {
	Name          string
	DataType      string
	FieldType     FieldType
	FieldKey      string
	Validator     *string
	ValidaotrFunc *Function
	LoadFromDB    *string // If not nil, the field will be loaded from the database using the given string as WHERE clause
	NotEmpty      bool
}

type Request struct {
	StructName string
	Route      string
	Method     Method
	Fields     []Field
	Handler    *Function
}

type Function struct {
	Name    string
	Params  map[string]string
	Returns []string
}

type RequestsFile struct {
	FileName  string
	Package   string
	Imports   map[string]string
	Requests  []Request
	Functions []Function
}

func (t FieldType) Priority() int {
	switch t {
	case FieldHeader:
		return 0
	case FieldPath:
		return 1
	case FieldQuery:
		return 2
	case FieldBody:
		return 3
	case FieldForm:
		return 4
	default:
		return 5
	}
}

func (t FieldType) SimpleRetriever() string {
	switch t {
	case FieldHeader:
		return "Get"
	case FieldPath:
		return "Params"
	case FieldQuery:
		return "Query"
	case FieldForm:
		return "FormValue"
	default:
		return ""
	}
}
