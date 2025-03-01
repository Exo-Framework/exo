package gen

import (
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/exo-framework/exo/common"
)

// Generator is a struct that holds the information about the packages and the requests files used for glue code generation.
type Generator struct {
	packages map[string][]RequestsFile
	rc       map[string]string
	module   string
}

// NewGenerator creates a new Generator struct.
func NewGenerator() *Generator {
	gomod, err := os.ReadFile("go.mod")
	if err != nil {
		panic(err)
	}

	for _, line := range strings.Split(string(gomod), "\n") {
		if strings.HasPrefix(line, "module ") {
			return &Generator{
				packages: make(map[string][]RequestsFile),
				rc:       common.LoadRuntimeConfig(),
				module:   strings.TrimPrefix(line, "module "),
			}
		}
	}

	panic("module not found in go.mod")
}

// Generate generates the glue code go files for the exo framework.
func (g *Generator) Generate() error {
	for dir, files := range g.packages {
		if len(files) == 0 {
			continue
		}

		if err := g.generatePackage(dir, files[0].Package, files); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) generatePackage(dir, pkg string, files []RequestsFile) error {
	indexFile := jen.NewFile(pkg)
	indexFile.PackageComment("Code generated by exo. DO NOT EDIT.")

	registers := []jen.Code{}

	for _, reqFile := range files {
		file := jen.NewFile(reqFile.Package)
		file.PackageComment("Code generated by exo. DO NOT EDIT.")

		for _, req := range reqFile.Requests {
			file.Add(g.generateHandler(req))

			registers = append(registers, jen.Id("r").Dot(string(req.Method)).Call(jen.Lit(req.Route), jen.Id("exog_"+req.Handler.Name)))
		}

		if err := file.Save(strings.Replace(reqFile.FileName, ".go", "_gen.go", 1)); err != nil {
			return err
		}
	}

	indexFile.Add(
		jen.Func().Id("RegisterRoutes").Params(
			jen.Id("r").Op("*").Qual("github.com/gofiber/fiber/v2", "App"),
		).Block(
			registers...,
		))

	return indexFile.Save(path.Join(dir, "index_gen.go"))
}

func (g *Generator) generateHandler(req Request) jen.Code {
	mainCodes := []jen.Code{}

	for _, field := range req.Fields {
		codes := []jen.Code{}

		if field.FieldType != FieldBody {
			rvPrefix := "raw_"
			if field.DataType == "string" {
				rvPrefix = "q_"
			}

			varname := rvPrefix + field.Name
			codes = append(codes, jen.Id(varname).Op(":=").Id("c").Dot(field.FieldType.SimpleRetriever()).Call(jen.Lit(field.FieldKey)))

			if field.Validator != nil {
				codes = append(codes,
					jen.If(
						jen.Id(varname+"_validator_errmsg").Op(":=").Id(*field.Validator).Call(jen.Id(varname)),
						jen.Id(varname+"_validator_errmsg").Op("!=").Lit(""),
					).Block(
						jen.Return(
							jen.Id("c").Dot("Status").Call(jen.Lit(400)).Dot("SendString").Call(jen.Id(varname+"_validator_errmsg")),
						),
					))
			} else if field.NotEmpty && field.DataType == "string" {
				codes = append(codes,
					jen.If(
						jen.Id(varname).Op("==").Lit(""),
					).Block(
						jen.Return(
							jen.Id("c").Dot("Status").Call(jen.Lit(400)).Dot("SendString").Call(jen.Lit(field.Name+" must not be empty")),
						),
					))
			}

			if field.DataType == "uuid.UUID" {
				codes = append(codes,
					jen.List(jen.Id("q_"+field.Name), jen.Id("q_"+field.Name+"_err")).Op(":=").Qual("github.com/google/uuid", "Parse").Call(jen.Id(varname)),
					jen.If(
						jen.Id("q_"+field.Name+"_err").Op("!=").Nil(),
					).Block(
						jen.Return(
							jen.Id("c").Dot("Status").Call(jen.Lit(400)).Dot("SendString").Call(jen.Id("q_"+field.Name+"_err").Dot("Error").Call()),
						),
					),
				)
			} else if field.DataType == "bool" {
				codes = append(codes,
					jen.List(jen.Id("q_"+field.Name), jen.Id("q_"+field.Name+"_err")).Op(":=").Qual("strconv", "ParseBool").Call(jen.Id(varname)),
					jen.If(
						jen.Id("q_"+field.Name+"_err").Op("!=").Nil(),
					).Block(
						jen.Return(
							jen.Id("c").Dot("Status").Call(jen.Lit(400)).Dot("SendString").Call(jen.Id("q_"+field.Name+"_err").Dot("Error").Call()),
						),
					),
				)
			} else if field.DataType == "int" {
				codes = append(codes,
					jen.List(jen.Id("q_"+field.Name), jen.Id("q_"+field.Name+"_err")).Op(":=").Qual("strconv", "Atoi").Call(jen.Id(varname)),
					jen.If(
						jen.Id("q_"+field.Name+"_err").Op("!=").Nil(),
					).Block(
						jen.Return(
							jen.Id("c").Dot("Status").Call(jen.Lit(400)).Dot("SendString").Call(jen.Id("q_"+field.Name+"_err").Dot("Error").Call()),
						),
					),
				)
			} else if strings.HasPrefix(field.DataType, "int") {
				b, err := strconv.Atoi(strings.TrimPrefix(field.DataType, "int"))
				if err != nil || (b != 8 && b != 16 && b != 32 && b != 64) {
					panic(ErrInvalidNumberBits)
				}
				codes = append(codes,
					jen.List(jen.Id("q_"+field.Name), jen.Id("q_"+field.Name+"_err")).Op(":=").Qual("strconv", "ParseInt").Call(jen.Id(varname), jen.Lit(10), jen.Lit(b)),
					jen.If(
						jen.Id("q_"+field.Name+"_err").Op("!=").Nil(),
					).Block(
						jen.Return(
							jen.Id("c").Dot("Status").Call(jen.Lit(400)).Dot("SendString").Call(jen.Id("q_"+field.Name+"_err").Dot("Error").Call()),
						),
					),
				)
			} else if strings.HasPrefix(field.DataType, "float") {
				b, err := strconv.Atoi(strings.TrimPrefix(field.DataType, "float"))
				if err != nil || (b != 32 && b != 64) {
					panic(ErrInvalidNumberBits)
				}
				codes = append(codes,
					jen.List(jen.Id("q_"+field.Name), jen.Id("q_"+field.Name+"_err")).Op(":=").Qual("strconv", "ParseFloat").Call(jen.Id(varname), jen.Lit(b)),
					jen.If(
						jen.Id("q_"+field.Name+"_err").Op("!=").Nil(),
					).Block(
						jen.Return(
							jen.Id("c").Dot("Status").Call(jen.Lit(400)).Dot("SendString").Call(jen.Id("q_"+field.Name+"_err").Dot("Error").Call()),
						),
					),
				)
			} else if strings.HasPrefix(field.DataType, "uint") {
				b, err := strconv.Atoi(strings.TrimPrefix(field.DataType, "uint"))
				if err != nil || (b != 8 && b != 16 && b != 32 && b != 64) {
					panic(ErrInvalidNumberBits)
				}
				codes = append(codes,
					jen.List(jen.Id("q_"+field.Name), jen.Id("q_"+field.Name+"_err")).Op(":=").Qual("strconv", "ParseUint").Call(jen.Id(varname), jen.Lit(10), jen.Lit(b)),
					jen.If(
						jen.Id("q_"+field.Name+"_err").Op("!=").Nil(),
					).Block(
						jen.Return(
							jen.Id("c").Dot("Status").Call(jen.Lit(400)).Dot("SendString").Call(jen.Id("q_"+field.Name+"_err").Dot("Error").Call()),
						),
					),
				)
			} else if field.LoadFromDB != nil {
				ptr := "&"
				if !strings.HasPrefix(field.DataType, "*") {
					ptr = ""
				}
				codes = append(codes,
					jen.Id("q_"+field.Name).Op(":=").Op(ptr).Id(field.DataType).Values(),
					jen.Id("q_"+field.Name+"_err").Op(":=").Qual(g.getDbPkg(), "DB").Dot("Where").Call(jen.Lit(*field.LoadFromDB+"=?"), jen.Id("raw_"+field.Name)).Dot("First").Call(jen.Op("&").Id("q_"+field.Name)).Dot("Error"),
					jen.If(
						jen.Id("q_"+field.Name+"_err").Op("==").Qual("gorm.io/gorm", "ErrRecordNotFound"),
					).Block(
						jen.Return(
							jen.Id("c").Dot("Status").Call(jen.Lit(404)).Dot("SendString").Call(jen.Lit(field.Name+" not found")),
						),
					),
					jen.If(
						jen.Id("q_"+field.Name+"_err").Op("!=").Nil(),
					).Block(
						jen.Return(
							jen.Id("q_"+field.Name+"_err"),
						),
					),
				)
			}
		} else {
			ptr := "&"
			if !strings.HasPrefix(field.DataType, "*") {
				ptr = ""
			}
			codes = append(codes,
				jen.Id("q_"+field.Name).Op(":=").Op(ptr).Id(field.DataType).Values(),
				jen.If(
					jen.Id("q_"+field.Name+"_err").Op(":=").Id("c").Dot("BodyParser").Call(jen.Op("&").Id("q_"+field.Name)),
					jen.Id("q_"+field.Name+"_err").Op("!=").Nil(),
				).Block(
					jen.Return(
						jen.Id("c").Dot("Status").Call(jen.Lit(400)).Dot("SendString").Call(jen.Id("q_"+field.Name+"_err").Dot("Error").Call()),
					),
				),
			)
		}

		mainCodes = append(mainCodes, codes...)
	}

	mainCodes = append(mainCodes, jen.Id("req").Op(":=").Id(req.StructName).Values(
		jen.DictFunc(func(d jen.Dict) {
			d[jen.Id(string(req.Method))] = jen.Qual("github.com/exo-framework/exo", string(req.Method)).Values(
				jen.Id("Ctx").Op(":").Id("c"),
			)

			for _, field := range req.Fields {
				d[jen.Id(field.Name)] = jen.Id("q_" + field.Name)
			}
		}),
	))

	finish := func() jen.Code {
		return jen.Func().Id("exog_" + req.Handler.Name).Params(
			jen.Id("c").Op("*").Qual("github.com/gofiber/fiber/v2", "Ctx"),
		).Error().Block(
			mainCodes...,
		)
	}

	if len(req.Handler.Returns) == 0 {
		mainCodes = append(mainCodes,
			jen.Id(req.Handler.Name).Call(jen.Id("req")),
			jen.Return(
				jen.Id("c").Dot("SendStatus").Call(jen.Lit(204)),
			))

		return finish()
	}

	returns := map[string]string{} // type -> name
	hadContentRet := false

	mainCodes = append(mainCodes,
		jen.ListFunc(func(l *jen.Group) {
			for i, ret := range req.Handler.Returns {
				if ret == "string" || ret == "[]byte" || ret == "interface{}" || ret == "any" {
					if hadContentRet {
						l.Id("_")
						continue
					}

					hadContentRet = true
				}

				rname := "r_" + strconv.Itoa(i)
				returns[ret] = rname

				l.Id(rname)
			}
		}).Op(":=").Id(req.Handler.Name).Call(jen.Id("req")))

	if rErrName, ok := returns["error"]; ok {
		mainCodes = append(mainCodes,
			jen.If(jen.Id(rErrName).Op("!=").Nil()).Block(
				jen.Return(jen.Id(rErrName)),
			))

		delete(returns, "error") // handled
	}

	extractStatus := func() (string, bool) {
		n := ""
		t2 := ""
		b := false

		for t, name := range returns {
			if strings.HasPrefix(t, "int") {
				t2 = t
				n = name
				b = true
				break
			}
		}

		if b {
			delete(returns, t2)
		}

		return n, b
	}

	hasContent := func() bool {
		for t := range returns {
			if t == "interface{}" || t == "any" || t == "string" || t == "[]byte" {
				return true
			}
		}

		return false
	}

	rStatusName, hasStatus := extractStatus()
	if hasStatus {
		if hasContent() {
			mainCodes = append(mainCodes,
				jen.Id("c").Dot("Status").Call(jen.Id(rStatusName)),
			)
		} else {
			mainCodes = append(mainCodes,
				jen.Return(jen.Id("c").Dot("SendStatus").Call(jen.Id(rStatusName))),
			)
		}
	}

	hasReturned := false

	for t, name := range returns {
		if t == "interface{}" || t == "any" {
			mainCodes = append(mainCodes,
				jen.Return(jen.Id("c").Dot("JSON").Call(jen.Id(name))),
			)

			hasReturned = true
			break
		} else if t == "string" {
			mainCodes = append(mainCodes,
				jen.Return(jen.Id("c").Dot("SendString").Call(jen.Id(name))),
			)

			hasReturned = true
			break
		} else if t == "[]byte" {
			mainCodes = append(mainCodes,
				jen.Return(jen.Id("c").Dot("Send").Call(jen.Id(name))),
			)

			hasReturned = true
			break
		}
	}

	if !hasReturned {
		mainCodes = append(mainCodes,
			jen.Return(jen.Id("c").Dot("SendStatus").Call(jen.Lit(204))),
		)
	}

	return finish()
}

func (g *Generator) getDbPkg() string {
	pkg, ok := g.rc["DB_PACKAGE"]
	if !ok {
		pkg = "db"
	}
	return g.module + "/" + pkg
}
