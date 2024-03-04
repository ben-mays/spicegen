package internal

import (
	_ "embed"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path"
	"sort"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

//go:embed types.text
var typestmptext string

//go:embed client.text
var clienttmptext string

//go:embed resource.text
var resourcetmptext string

func genFormattedSource(context any, templateTxt, outputDir, filename string) {
	fmap := map[string]any{"ToUpper": strings.ToUpper, "ToCamel": strcase.ToCamel}
	tmpl, err := template.New("").Funcs(fmap).Parse(templateTxt)
	if err != nil {
		panic(fmt.Errorf("Error parsing template: %s", err.Error()))
	}
	buf := &strings.Builder{}
	tmpl.Execute(buf, context)
	s := buf.String()
	res, err := format.Source([]byte(s))
	if err != nil {
		fmt.Println(s)
		panic(fmt.Errorf("Error formatting source: %s", err.Error()))
	}
	err = os.Mkdir(outputDir, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		panic(fmt.Errorf("Error creating directory: %s", err.Error()))
	} else {
		err = nil
	}
	err = os.WriteFile(path.Join(outputDir, filename), res, 0644)
	if err != nil {
		panic(fmt.Errorf("Error writing file: %s", err.Error()))
	}
}

func SortedKeys[T any](anyMap map[string]T) []string {
	keys := make([]string, 0, len(anyMap))
	for k := range anyMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Returns an array of values from a map, sorted by the keys
func SortedMap[T any](anyMap map[string]T) []T {
	keys := SortedKeys[T](anyMap)
	values := make([]T, 0, len(keys))
	for _, k := range keys {
		values = append(values, anyMap[k])
	}
	return values
}

func GenTypes(state Schema, outputDir, outputFileName, packageName string, interfaceName string, resourceImportPath string) {
	resources := SortedMap(state.Resources)
	for i, rsc := range resources {
		rsc.PermissionsArray = SortedMap(rsc.Permissions)
		rsc.RelationsArray = SortedMap(rsc.Relations)
		resources[i] = rsc
	}
	genFormattedSource(struct {
		PackageName   string
		InterfaceName string
		ImportPath    string
		Resources     []Resource
	}{PackageName: packageName, InterfaceName: interfaceName, ImportPath: resourceImportPath, Resources: resources}, typestmptext, outputDir, outputFileName)
}

func GenClient(state Schema, outputDir, outputFileName, packageName string, clientName string, interfaceName string, resourceImportPath string) {
	resources := SortedMap(state.Resources)
	for i, rsc := range resources {
		rsc.PermissionsArray = SortedMap(rsc.Permissions)
		rsc.RelationsArray = SortedMap(rsc.Relations)
		resources[i] = rsc
	}
	genFormattedSource(struct {
		PackageName   string
		ClientName    string
		InterfaceName string
		ImportPath    string
		Resources     []Resource
	}{PackageName: packageName, ClientName: clientName, InterfaceName: interfaceName, ImportPath: resourceImportPath, Resources: resources}, clienttmptext, outputDir, outputFileName)
}

func GenResource(rsc Resource, outputDir, packageName string) {
	rsc.PermissionsArray = SortedMap(rsc.Permissions)
	rsc.RelationsArray = SortedMap(rsc.Relations)
	genFormattedSource(struct {
		PackageName string
		Resource    Resource
	}{PackageName: packageName, Resource: rsc}, resourcetmptext, path.Join(outputDir, packageName), fmt.Sprintf("%s.go", rsc.Name))
}
