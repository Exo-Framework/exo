package gentest

import (
	"github.com/exo-framework/exo"
	"github.com/google/uuid"
)

type SomeDbModel struct{}

type GetTest struct {
	exo.Get       `route:"/test/:id/:id2"` // this will generate a GET method for the /test/:id/:id2 route
	Id            int                      `path:"id"`                                 // this will load the path parameter "id" into the Id field
	Id2           uuid.UUID                `path:""`                                   // this will load the path parameter "id2" into the Id2 field. Omitting the name will use the field name in camel case notation
	Name          string                   `query:"name"`                              // this will load the query parameter "name" into the Name field
	Name2         string                   `query:""`                                  // this will load the query parameter "name2" into the Name2 field. Omitting the name will use the field name in camel case notation
	Authorization string                   `header:"Authorization"`                    // this will load the header "Authorization" into the Authorization field
	Validator     string                   `header:"Validator" validate:"onValidator"` // this will load the header "Validator" into the Validator field and validate it using the onValidator function
	Dto           GetTestDto               `body:""`                                   // this will load the body into the Dto field
	Form          string                   `form:""`                                   // this will load the form parameter "form" into the Form field
	FormNamed     string                   `form:"form_named"`                         // this will load the form parameter "form_named" into the FormNamed field
	SomeDbModel   SomeDbModel              `path:"id" db:"id"`                         // this will load the path parameter "id" and uses it as WHERE against the SomeDbModel table to load the SomeDbModel field. If not found is returned, 404 is returned. If db is left empty, the default primary key is used
}

type GetTestDto struct {
	Id int `json:"id"` // this will load the json field "id" into the Id field
}

// Handler can return any tuple combinations (or alone) of the following data types:
// - error: if an error is returned, the error handler will be called
// - int, int8, int16, int32, int64: the int as as status code will be set
// - string: the string as plain text response
// - []byte: the byte slice as binary response/file
// - interface{}, any: the interface{} will be serialized as JSON response
func getTest(GetTest) (string, error) {
	return "", nil
}

func onValidator(string) string {
	return "" // return an empty string if the value is valid, otherwise the error message which should be appended to the 400 response
}
