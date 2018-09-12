package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	filename := flag.String("f", "", "*.pb.go path")
	flag.Parse()

	if *filename == "" {
		fmt.Println("f is empty")
		return
	}

	fset := token.NewFileSet()
	pkgPath := strings.TrimSuffix(*filename, fmt.Sprintf("/%v", filepath.Base(*filename)))
	f, err := parser.ParseFile(fset, filepath.Join(*filename, fmt.Sprintf("%v.pb.go", filepath.Base(pkgPath))), nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	var (
		pkgName    = ""
		intfName   = ""
		methodList []*ast.Field
	)

	serviceServer := false
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.File:
			pkgName = x.Name.Name
		case *ast.TypeSpec:
			intfName = x.Name.Name
			if intfName == "ServiceServer" {
				serviceServer = true
			}
		case *ast.InterfaceType:
			methodList = x.Methods.List
		}
		return !serviceServer
	})
	genService(*filename, pkgName, methodList)
}

func genService(path, pkgName string, methodList []*ast.Field) {
	fileName := filepath.Join(path, "service.go")
	if _, err := os.Stat(fileName); !os.IsNotExist(err) {
		os.Remove(fileName)
	}
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}

	io.WriteString(f, fmt.Sprintf(`
package %v

import "context"

type Service interface {
`, pkgName))
	for _, m := range methodList {
		thisfunc := m.Type.(*ast.FuncType)
		if m.Doc != nil {
			for _, v := range m.Doc.List {
				io.WriteString(f, fmt.Sprintf(`%v
`, v.Text))
			}
		}

		secondArgs := thisfunc.Params.List[1].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		firstResp := thisfunc.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
		methodName := m.Names[0].Name

		io.WriteString(f, fmt.Sprintf(`%v(context.Context, *%v) (*%v, error)
`, methodName, secondArgs, firstResp))
	}

	io.WriteString(f, "}")
	f.Close()
	execCommand("gofmt", "-w", fileName)
}

func execCommand(name string, args ...string) {
	if err := exec.Command(name, args...).Run(); err != nil {
		fmt.Println("execCommand Error", err)
	}
}
