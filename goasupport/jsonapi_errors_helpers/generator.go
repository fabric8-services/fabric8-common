package helperfunctions

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/goadesign/goa/design"
	"github.com/goadesign/goa/goagen/codegen"
	errs "github.com/pkg/errors"
)

// Generate adds method to support conditional queries
func Generate() ([]string, error) {
	var (
		ver    string
		outDir string
	)
	set := flag.NewFlagSet("app", flag.PanicOnError)
	set.String("design", "", "") // Consume design argument so Parse doesn't complain
	set.StringVar(&ver, "version", "", "")
	set.StringVar(&outDir, "out", "", "")
	set.Parse(os.Args[2:])
	for _, arg := range os.Args {
		fmt.Println(arg)
	}
	// First check compatibility
	if err := codegen.CheckVersion(ver); err != nil {
		return nil, err
	}
	// return writeFunctions(design.Design, outDir)
	files := make([]string, 0)
	// the `jsonapi_utility.txt` file is in the same directory as this function itself
	for file, title := range map[string]string{
		"error_handler.go.txt":                          "helper functions to handle errors",
		"jsonapi_errors_stringer.go.txt":                "stringer functions for JSONAPI Error(s)",
		"jsonapi_errors_converter.go.txt":               "helper functions to convert to JSONAPI Errors",
		"jsonapi_errors_converter_blackbox_test.go.txt": "tests for the helper functions to convert to JSONAPI Errors",
	} {
		result, err := writeFile(file, title, design.Design, outDir)
		if err != nil {
			return []string{}, err
		}
		files = append(files, result)
	}
	return files, nil
}

func writeFile(sourceFile, title string, api *design.APIDefinition, outDir string) (string, error) {
	var caller string
	var ok bool
	if _, caller, _, ok = runtime.Caller(1); !ok {
		return "", errs.Errorf("failed to generate the JSON-API Errors helpers")
	}

	// looks-up the .go source file in this directory, parse it and generates the output file
	// in the target directory
	fset := token.NewFileSet() // positions are relative to fset
	// read the source file
	content, err := ioutil.ReadFile(filepath.Join(filepath.Dir(caller), sourceFile))
	if err != nil {
		return "", errs.Wrapf(err, "failed to generate the JSON-API Errors helper from %s", sourceFile)
	}
	f, err := parser.ParseFile(fset, sourceFile, content, parser.ImportsOnly)
	if err != nil {
		return "", errs.Wrapf(err, "failed to generate the JSON-API Errors helper from %s", sourceFile)
	}

	// computes the name of the file by removing all extensions, then adding `.go`
	targetFile := sourceFile
	for {
		ext := filepath.Ext(targetFile)
		if ext == "" {
			break
		}
		targetFile = strings.TrimRight(targetFile, ext)
	}
	targetFile = fmt.Sprintf("%s.go", targetFile)
	ctxFile := filepath.Join(outDir, targetFile)
	ctxWr, err := codegen.SourceFileFor(ctxFile)
	if err != nil {
		return "", errs.Wrapf(err, "failed to generate the JSON-API Errors helper from %s", sourceFile)
	}
	imports := make([]*codegen.ImportSpec, len(f.Imports))
	for i, s := range f.Imports {
		// fmt.Println(s.Path.Value)
		if s.Name != nil {
			imports[i] = codegen.NewImport(s.Name.String(), strings.Trim(s.Path.Value, `"`))
		} else {
			imports[i] = codegen.SimpleImport(strings.Trim(s.Path.Value, `"`))
		}
	}
	ctxWr.WriteHeader(title, "app", imports)
	endOfImports := f.Decls[0].End()
	ctxWr.Write(content[endOfImports:])

	return "", nil
}

// writeJSONAPIErrorConverterFile creates the `jsonapi_errors_converter.go` file.
func writeJSONAPIErrorConverterFile(api *design.APIDefinition, outDir string) ([]string, error) {
	ctxFile := filepath.Join(outDir, "jsonapi_errors_converter.go")
	ctxWr, err := codegen.SourceFileFor(ctxFile)
	if err != nil {
		panic(err) // bug
	}
	title := fmt.Sprintf("%s: functions for JSONAPI Errors - See vendor/fabric8-services/fabric8-common/goasupport/jsonapi_errors_stringer/generator.go", api.Context())
	imports := []*codegen.ImportSpec{
		codegen.SimpleImport("fmt"),
		codegen.SimpleImport("github.com/davecgh/go-spew/spew"),
		codegen.SimpleImport("context"),
		codegen.SimpleImport("net/http"),
		codegen.SimpleImport("reflect"),
		codegen.SimpleImport("strconv"),
		codegen.SimpleImport("github.com/fabric8-services/fabric8-common/errors"),
		codegen.SimpleImport("github.com/fabric8-services/fabric8-common/log"),
		codegen.SimpleImport("github.com/fabric8-services/fabric8-common/sentry"),
		codegen.SimpleImport("github.com/goadesign/goa"),
		codegen.NewImport("errs", "github.com/pkg/errors"),
	}
	ctxWr.WriteHeader(title, "app", imports)
	var f string
	var ok bool
	if _, f, _, ok = runtime.Caller(1); !ok {
		return nil, errs.Wrapf(err, "failed to generate the JSON-API Errors helpers")
	}
	// the `jsonapi_utility.txt` file is in the same directory as this function itself
	f = filepath.Join(filepath.Dir(f), "jsonapi_utility.txt")
	fmt.Printf("including content from %s\n", f)
	body, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to generate the JSON-API Errors helpers")
	}
	if _, err := ctxWr.Write(body); err != nil {
		return nil, errs.Wrapf(err, "failed to generate the JSON-API Errors helpers")
	}
	return []string{ctxFile}, nil
}

// writeJSONAPIErrorHandlerFile creates the `jsonapi_errors_handler.go` file.
func writeJSONAPIErrorHandlerFile(api *design.APIDefinition, outDir string) ([]string, error) {
	ctxFile := filepath.Join(outDir, "jsonapi_errors_converter.go")
	ctxWr, err := codegen.SourceFileFor(ctxFile)
	if err != nil {
		panic(err) // bug
	}
	title := fmt.Sprintf("%s: String functions for JSONAPI Errors - See vendor/fabric8-services/fabric8-common/goasupport/jsonapi_errors_stringer/generator.go", api.Context())
	imports := []*codegen.ImportSpec{
		codegen.SimpleImport("fmt"),
		codegen.SimpleImport("github.com/davecgh/go-spew/spew"),
		codegen.SimpleImport("context"),
		codegen.SimpleImport("net/http"),
		codegen.SimpleImport("reflect"),
		codegen.SimpleImport("strconv"),
		codegen.SimpleImport("github.com/fabric8-services/fabric8-common/errors"),
		codegen.SimpleImport("github.com/fabric8-services/fabric8-common/log"),
		codegen.SimpleImport("github.com/fabric8-services/fabric8-common/sentry"),
		codegen.SimpleImport("github.com/goadesign/goa"),
		codegen.NewImport("errs", "github.com/pkg/errors"),
	}
	ctxWr.WriteHeader(title, "app", imports)
	var f string
	var ok bool
	if _, f, _, ok = runtime.Caller(1); !ok {
		return nil, errs.Wrapf(err, "failed to generate the JSON-API Errors helpers")
	}
	// the `jsonapi_utility.txt` file is in the same directory as this function itself
	f = filepath.Join(filepath.Dir(f), "jsonapi_utility.txt")
	fmt.Printf("including content from %s\n", f)
	body, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to generate the JSON-API Errors helpers")
	}
	if _, err := ctxWr.Write(body); err != nil {
		return nil, errs.Wrapf(err, "failed to generate the JSON-API Errors helpers")
	}
	return []string{ctxFile}, nil
}
