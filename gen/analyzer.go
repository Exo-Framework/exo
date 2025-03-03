package gen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Analyze analyzes the given directory for packages and requests files.
func (g *Generator) Analyze(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	reqFiles := make([]RequestsFile, 0, len(files))

	for _, file := range files {
		if file.IsDir() {
			if err := g.Analyze(filepath.Join(dir, file.Name())); err != nil {
				return err
			}
			continue
		}

		if strings.HasSuffix(file.Name(), ".go") && !strings.HasSuffix(file.Name(), "_gen.go") {
			if err := g.analyzeFile(filepath.Join(dir, file.Name()), &reqFiles); err != nil {
				return err
			}
		}
	}

	if len(reqFiles) > 0 {
		g.packages[dir] = reqFiles
	}

	return nil
}

func (g *Generator) analyzeFile(filePath string, files *[]RequestsFile) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	reqFile := RequestsFile{
		FileName:  filePath,
		Package:   node.Name.Name,
		Imports:   make(map[string]string),
		Requests:  []Request{},
		Functions: []Function{},
	}

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				for _, spec := range d.Specs {
					importSpec := spec.(*ast.ImportSpec)
					importPath := strings.Trim(importSpec.Path.Value, `"`)
					importName := ""
					if importSpec.Name != nil {
						importName = importSpec.Name.Name
					}
					reqFile.Imports[importPath] = importName
				}
			} else if d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					typeSpec := spec.(*ast.TypeSpec)
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						g.extractRequestStruct(typeSpec.Name.Name, structType, &reqFile)
					}
				}
			}

		case *ast.FuncDecl:
			g.extractFunction(d, &reqFile)
		}
	}

	for k, req := range reqFile.Requests {
		for i, field := range req.Fields {
			if field.Validator != nil {
				for _, fn := range reqFile.Functions {
					if fn.Name == *field.Validator {
						field.ValidaotrFunc = &fn
						req.Fields[i] = field
						break
					}
				}

				if field.ValidaotrFunc == nil {
					return errors.Join(ErrFunctionNotFound, fmt.Errorf("validator: for field %s in struct %s", field.Name, req.StructName))
				}
			}
		}

		for _, fn := range reqFile.Functions {
			for _, param := range fn.Params {
				if param == req.StructName {
					req.Handler = &fn
					reqFile.Requests[k] = req
					break
				}
			}
		}

		if req.Handler == nil {
			return errors.Join(ErrFunctionNotFound, fmt.Errorf("handler: for struct %s", req.StructName))
		}

		if len(req.Handler.Params) != 1 {
			return ErrHandlerIllegalSignature
		}

		for _, ret := range req.Handler.Returns {
			if !g.isAllowedReturnType(ret) {
				return ErrHandlerIllegalSignature
			}
		}
	}

	*files = append(*files, reqFile)
	return nil
}

func (g *Generator) extractRequestStruct(name string, structType *ast.StructType, reqFile *RequestsFile) {
	var req Request
	req.StructName = name
	req.Route = ""
	req.Method = ""
	req.Fields = []Field{}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			if ident, ok := field.Type.(*ast.SelectorExpr); ok {
				if x, ok := ident.X.(*ast.Ident); ok && x.Name == "exo" {
					req.Method = Method(ident.Sel.Name)

					tagValue := ""
					if field.Tag != nil {
						tagValue = field.Tag.Value
						tagValue = strings.Trim(tagValue, "`")
					}

					tagParts := strings.Split(tagValue, " ")
					for i := 0; i < len(tagParts); i++ {
						tag := tagParts[i]
						if !strings.HasPrefix(tag, "route:") {
							continue
						}

						route := strings.TrimPrefix(tag, "route:")
						route = strings.TrimPrefix(route, `"`)
						route = strings.TrimSuffix(route, `"`)

						req.Route = route
					}
				}
			}
			continue
		}

		fieldName := field.Names[0].Name
		fieldType := ""
		if ident, ok := field.Type.(*ast.Ident); ok {
			fieldType = ident.Name
		} else if star, ok := field.Type.(*ast.StarExpr); ok {
			if ident, ok := star.X.(*ast.Ident); ok {
				fieldType = "*" + ident.Name
			}
		} else if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				fieldType = x.Name + "." + sel.Sel.Name
			}
		} else if s, ok := field.Type.(*ast.StructType); ok {
			fieldType = "struct {"
			for _, field := range s.Fields.List {
				if len(field.Names) == 0 {
					continue
				}

				fieldName := field.Names[0].Name
				fieldType += fmt.Sprintf(" %s %s", fieldName, field.Type)
				if field.Tag != nil {
					fieldType += fmt.Sprintf(" %s", field.Tag.Value)
				}

				fieldType += ";"
			}
			fieldType += " }"
		}

		tagValue := ""
		if field.Tag != nil {
			tagValue = field.Tag.Value
			tagValue = strings.Trim(tagValue, "`")
		}

		fieldTypeEnum := FieldType("")
		tagParts := strings.Split(tagValue, " ")
		fieldKey := ""
		notEmpty := false

		var fromDbClause *string
		var validator *string

		for i := 0; i < len(tagParts); i++ {
			tag := tagParts[i]
			pair := strings.Split(tag, ":")

			tagKey := pair[0]
			tagVal := pair[1]

			tagVal = strings.TrimPrefix(tagVal, `"`)
			tagVal = strings.TrimSuffix(tagVal, `"`)

			switch tagKey {
			case "path":
				fieldTypeEnum = FieldPath
				fieldKey = tagVal
			case "query":
				fieldTypeEnum = FieldQuery
				fieldKey = tagVal
			case "header":
				fieldTypeEnum = FieldHeader
				fieldKey = tagVal
			case "body":
				fieldTypeEnum = FieldBody
				fieldKey = tagVal
			case "form":
				fieldTypeEnum = FieldForm
				fieldKey = tagVal
			case "db":
				fromDbClause = &tagVal
			case "validate":
				if strings.EqualFold(tagVal, "notempty") {
					notEmpty = true
					continue
				}

				validator = &tagVal
			}
		}

		if fieldKey == "" {
			fieldKey = strings.ToLower(fieldName[:1]) + fieldName[1:]
		}

		req.Fields = append(req.Fields, Field{
			Name:       fieldName,
			DataType:   fieldType,
			FieldType:  fieldTypeEnum,
			FieldKey:   fieldKey,
			LoadFromDB: fromDbClause,
			Validator:  validator,
			NotEmpty:   notEmpty,
		})
	}

	sort.Slice(req.Fields, func(i, j int) bool {
		return req.Fields[i].FieldType.Priority() < req.Fields[j].FieldType.Priority()
	})

	if req.Method != "" {
		reqFile.Requests = append(reqFile.Requests, req)
	}
}

func (g *Generator) extractFunction(fn *ast.FuncDecl, reqFile *RequestsFile) {
	if fn.Recv != nil {
		return
	}

	function := Function{
		Name:    fn.Name.Name,
		Params:  make(map[string]string),
		Returns: []string{},
	}

	uk := 0

	for _, param := range fn.Type.Params.List {
		paramType := ""
		if ident, ok := param.Type.(*ast.Ident); ok {
			paramType = ident.Name
		} else if star, ok := param.Type.(*ast.StarExpr); ok {
			if ident, ok := star.X.(*ast.Ident); ok {
				paramType = "*" + ident.Name
			}
		} else if sel, ok := param.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				paramType = x.Name + "." + sel.Sel.Name
			}
		}

		name := ""
		if len(param.Names) == 0 {
			name = fmt.Sprintf("uk_%d", uk)
			uk++
		} else {
			name = param.Names[0].Name
		}

		function.Params[name] = paramType
	}

	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			if ident, ok := result.Type.(*ast.Ident); ok {
				function.Returns = append(function.Returns, ident.Name)
			} else if star, ok := result.Type.(*ast.StarExpr); ok {
				if ident, ok := star.X.(*ast.Ident); ok {
					function.Returns = append(function.Returns, "*"+ident.Name)
				}
			} else if sel, ok := result.Type.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok {
					function.Returns = append(function.Returns, x.Name+"."+sel.Sel.Name)
				}
			}
		}
	}

	reqFile.Functions = append(reqFile.Functions, function)
}

func (g *Generator) isAllowedReturnType(ret string) bool {
	switch ret {
	case "error", "int", "int8", "int16", "int32", "int64", "string", "[]byte", "interface{}", "any", "exo.Serialize":
		return true
	}
	return false
}
