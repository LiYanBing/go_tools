package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	gateService  = "gate"
	innerService = "inner"
)

func main() {
	svr := flag.String("m", gateService, "gate or inner of module")
	name := flag.String("name", "gate", "the module name")
	flag.Parse()

	if *svr == "" || *name == "" {
		fmt.Println("please input valid m or name")
		return
	}

	curPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	newPath := path.Join(curPath, *name)
	dirs := []string{
		path.Join(newPath, "service"), //service dir
		path.Join(newPath, "models"),  //models dir
	}

	servicePath := path.Join(newPath, "service")
	files := []string{
		path.Join(servicePath, "service.go"),
		path.Join(newPath, "main.go"),
		path.Join(newPath, fmt.Sprintf("%v-local.conf", *name)),
		path.Join(newPath, fmt.Sprintf("%v-test.conf", *name)),
		path.Join(newPath, fmt.Sprintf("%v-prod.conf", *name)),
	}

	switch *svr {
	case gateService:
		dirs = append(dirs, path.Join(newPath, "static")) //static dir
		files = append(files, path.Join(servicePath, "router.go"))
		fallthrough

	case innerService:
		apiPath := path.Join(newPath, "api")
		dirs = append(dirs, apiPath) //API
		for _, dir := range dirs {
			err := makeDir(dir)
			if err != nil {
				panic(err)
			}
		}

		files = append(files, path.Join(apiPath, "compile.sh"))
		files = append(files, path.Join(apiPath, *name+".proto"))
		files = append(files, path.Join(apiPath, "service.go"))
		for _, file := range files {
			f, err := makeFile(file)
			if err != nil {
				panic(err)
			}
			writeFile(*name, apiPath, file, f)
		}
	default:
		fmt.Println("plase check you enter m type")
	}
}

func writeFile(moduName, apiPath, pathName string, file *os.File) {
	fileName := filepath.Base(pathName)

	if !strings.Contains(pathName, "api") {
		switch fileName {
		case "service.go":
			writeServiceFile(moduName, apiPath, pathName, file)
		case "main.go":
			writeMainFile(pathName, file)
		}
		return
	}

	switch fileName {
	case "compile.sh":
		writeCompileFile(file)
	case "service.go":
		writeApiServiceFile(moduName, pathName, file)
	}

	if filepath.Ext(pathName) == ".proto" {
		writeProtoFile(moduName, file)
	}
}

func writeCompileFile(file *os.File) {
	_, err := io.WriteString(file, "protoc *.proto --go_out=plugins=grpc:.\n"+
		"project_path=$(cd `dirname $0`; pwd)\n"+
		"`cd $GOPATH/bin\n"+
		"./proto_tool -f $project_path\n"+
		"`")
	if err != nil {
		panic(err)
	}
	file.Close()
}

func writeApiServiceFile(modu, filePath string, file *os.File) {
	ss := fmt.Sprintf(`package %v
type Service interface {
}`, modu)
	_, err := io.WriteString(file, ss)
	if err != nil {
		panic(err)
	}
	file.Close()
	execCommand("gofmt", "-w", filePath)
}

func writeProtoFile(moduName string, file *os.File) {
	ss := fmt.Sprintf(`syntax = "proto3";

package %v;

service Service {
    rpc HellWorld(HellWorldArgs) returns (HellWorldResp) {}
}

enum BrickZone {
    Invalid = 0;
    Homepage = 1;
    Channelpage = 2; 
    SubjectGrouppage = 3;
}

message HellWorldArgs {
  string name = 1;
  string email = 2;
  string phone= 3;
  BrickZone brick =4;
}

message HellWorldResp {
  int32 id = 1;
  bool success = 2;
}`, moduName)

	_, err := io.WriteString(file, ss)
	if err != nil {
		panic(err)
	}
	file.Close()
}

func writeServiceFile(moduName, apiPath, filePathName string, file *os.File) {
	apiPath = strings.TrimPrefix(apiPath, filepath.Join(os.Getenv("GOPATH"), "src"))
	apiPath = apiPath[1:]
	serviceName := "ServiceCloser"
	ss := fmt.Sprintf(`
package service

import (
	"%v"
	"github.com/go-kit/kit/log"
	"github.com/opentracing/opentracing-go"
)

type Conf struct {
	ConsulAddr   string            
}

type service struct {
}

type %v interface {
	%v.Service
	Close() error
}

func New(cfg *Conf, tracer opentracing.Tracer, logger log.Logger) %v {
	return &service{
	}
}

func (s *service) Close() error {
	panic("implement me")
}
`, apiPath, serviceName, moduName, serviceName)

	_, err := io.WriteString(file, ss)
	if err != nil {
		panic(err)
	}
	file.Close()
	execCommand("gofmt", "-w", filePathName)
}

func writeMainFile(filePathName string, file *os.File) {
	ss := `
package main

func main() {
	
}
`
	_, err := io.WriteString(file, ss)
	if err != nil {
		panic(err)
	}
	file.Close()
	execCommand("gofmt", "-w", filePathName)
}

func execCommand(name string, args ...string) {
	if err := exec.Command(name, args...).Run(); err != nil {
		fmt.Println("execCommand Error", err)
	}
}

func makeDir(d string) error {
	return os.MkdirAll(d, 0777)
}

func makeFile(f string) (*os.File, error) {
	return os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
}

func firstUpperCase(s string) string {
	if len(s) > 0 {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	return ""
}
