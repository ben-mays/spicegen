package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/authzed/spicedb/pkg/schemadsl/compiler"
	"github.com/ben-mays/spicegen/internal"
	"golang.org/x/exp/maps"
)

func main() {
	fs := flag.NewFlagSet("spicegen", flag.ContinueOnError)
	schemaPath := fs.String(
		"schema-file",
		"schema.text",
		"Optional. Path to schema file for generation. If none given, the tool will look for schema.text in the current directory.",
	)

	outputPath := fs.String(
		"output-path",
		"",
		"Optional. The file or directory to which the generated client will be written. If a directory is given, the output filename will be client.go. If no output is given, current directory is used.",
	)

	outputPackageName := fs.String(
		"output-package",
		"",
		"Optional. The package name of the generated client. This will default to the output directory name if not given.",
	)

	outputClientName := fs.String(
		"client-name",
		"Client",
		"Optional. The name of the client impl created by spicegen.",
	)

	outputInterfaceName := fs.String(
		"interface-name",
		"SpiceGenClient",
		"Optional. The name of the client interface created by spicegen.",
	)

	ignorePrefix := fs.String(
		"ignore-prefix",
		"",
		"Optional. A prefix string to match against permission/relation names to ignore. Used to avoid exposing implicit permissions.",
	)

	outputImportPath := fs.String(
		"import-path",
		"",
		"Required. The fully qualified module path for importing the generated client. e.x. github.com/ben-mays/spicegen/example",
	)

	skipClientGeneration := fs.Bool(
		"skip-client",
		false,
		"Optional. If present, will skip client generation and only generate types and permissions.",
	)

	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("Error parsing flags: %s", err.Error())
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("Error getting current directory: %s", err.Error())
		return
	}

	// Setup output path to PWD if not set
	if outputPath == nil || *outputPath == "" {
		outputPath = &wd
	}

	// Setup output package name
	if outputPackageName == nil || *outputPackageName == "" {
		base := path.Base(*outputPath)
		outputPackageName = &base
	}

	// if output path is relative, make it abs
	if !path.IsAbs(*outputPath) {
		newPath := path.Join(wd, *outputPath)
		outputPath = &newPath
	}

	// Setup output file
	outputFileName := "client.go"
	if path.Dir(*outputPath) != *outputPath {
		base := path.Base(*outputPath)
		if path.Ext(*outputPath) == ".go" {
			outputFileName = base
		}
	}

	if outputImportPath == nil || *outputImportPath == "" {
		fmt.Println("Flag `input-path` is required.")
		fs.Usage()
		return
	}

	generatedFilePath := path.Join(*outputPath, outputFileName)
	permissionPath := path.Join(*outputPath, "permissions")
	defer func() {
		if err != nil {
			fmt.Println(err)
			fmt.Println("failed to generate client, cleaning up.")
			os.Remove(generatedFilePath)
			os.RemoveAll(permissionPath)
		}
	}()

	fmt.Printf("reading schema file %s\n", *schemaPath)
	schematxt, err := os.ReadFile(*schemaPath)
	if err != nil {
		// try cwd
		cwd, _ := os.Getwd()
		schematxt, err = os.ReadFile(path.Join(cwd, *schemaPath))
		if err != nil {
			err = fmt.Errorf("Error reading schema file: %s", err.Error())
			return
		}
	}

	prefix := ""
	resp, err := compiler.Compile(compiler.InputSchema{SchemaString: string(schematxt)}, compiler.ObjectTypePrefix(prefix))
	if err != nil {
		err = fmt.Errorf("Error compiling schema file: %s", err.Error())
		return
	}

	// create permission directories
	err = os.Mkdir(permissionPath, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		err = fmt.Errorf("Error creating output directory: %s", err.Error())
		return
	} else {
		// reset errExist errors to nil
		err = nil
	}
	// generate client/resources
	state := internal.BuildSchema(resp)
	if ignorePrefix != nil && *ignorePrefix != "" {
		// delete ignored keys from state to avoid rendering them
		for _, resource := range state.Resources {
			for key := range resource.Permissions {
				if strings.HasPrefix(key, *ignorePrefix) {
					delete(resource.Permissions, key)
				}
			}
			for key := range resource.Relations {
				if strings.HasPrefix(key, *ignorePrefix) {
					delete(resource.Relations, key)
				}
			}
		}
	}

	// Sort everything
	resources := SortedMap(state.Resources)
	for i, rsc := range resources {
		resources[i].PermissionsArray = SortedMap(rsc.Permissions)
		resources[i].RelationsArray = SortedMap(rsc.Relations)
	}

	fmt.Printf("writing types to %s with packageName %s\n", path.Join(*outputPath, "types.go"), *outputPackageName)
	internal.GenTypes(maps.Values(state.Resources), *outputPath, "types.go", *outputPackageName, *outputInterfaceName, *outputImportPath)
	if !*skipClientGeneration {
		fmt.Printf("writing client to %s with packageName %s\n", path.Join(*outputPath, outputFileName), *outputPackageName)
		internal.GenClient(maps.Values(state.Resources), *outputPath, outputFileName, *outputPackageName, *outputClientName, *outputInterfaceName, *outputImportPath)
	}
	for _, rsc := range resources {
		internal.GenResource(rsc, permissionPath, rsc.Name)
	}
}

func SortedKeys[T any](anyMap map[string]T) []string {
	keys := make([]string, 0)
	for k := range anyMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Returns an array of values from a map, sorted by the keys
func SortedMap[T any](anyMap map[string]T) []T {
	keys := SortedKeys[T](anyMap)
	values := make([]T, len(keys))
	for i, k := range keys {
		values[i] = anyMap[k]
	}
	return values
}
