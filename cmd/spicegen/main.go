package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/authzed/spicedb/pkg/schemadsl/compiler"
	"github.com/ben-mays/spicegen/internal"
)

func main() {
	fs := flag.NewFlagSet("spicegen", flag.ContinueOnError)
	schemaPath := fs.String(
		"s",
		"schema.text",
		"Path to schema file for generation. If none given, the tool will look for schema.text in the current directory.",
	)

	outputPath := fs.String(
		"o",
		"",
		"The file or directory to which the generated client will be written. If a directory is given, the output filename will be client.go. If no output is given, current directory is used.",
	)

	outputPackageName := fs.String(
		"op",
		"",
		"The package name of the generated client. This will default to the output directory name if not given.",
	)

	outputImportPath := fs.String(
		"module-name",
		"",
		"Required. The base module name for wiring up imports. e.x. github.com/ben-mays/spicegen",
	)

	fs.Parse(os.Args[1:])

	// Setup output path to PWD if not set
	if outputPath == nil || *outputPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			err = fmt.Errorf("Error getting current directory: %s", err.Error())
			return
		}
		outputPath = &wd
	}

	// Setup output package name
	if outputPackageName == nil || *outputPackageName == "" {
		base := path.Base(*outputPath)
		outputPackageName = &base
	}

	// Setup output file
	outputFileName := "client.go"
	outputDir := *outputPath
	// output path is a file
	if path.Dir(*outputPath) != *outputPath {
		outputDir = path.Dir(*outputPath)
		base := path.Base(*outputPath)
		if strings.HasSuffix(base, ".go") {
			outputFileName = base
		} else {
			outputFileName = "client.go"
		}
	} else {
		outputDir = *outputPath
	}

	if outputImportPath == nil || *outputImportPath == "" {
		fmt.Println("Flag `module-name` is required.")
		fs.Usage()
		return
	}

	var err error
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
	resp, err := compiler.Compile(compiler.InputSchema{SchemaString: string(schematxt)}, &prefix)
	if err != nil {
		err = fmt.Errorf("Error compiling schema file: %s", err.Error())
		return
	}

	fmt.Printf("writing client to %s with packageName %s\n", path.Join(*outputPath, outputFileName), *outputPackageName)
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
	internal.GenClient(state, *outputPath, outputFileName, *outputPackageName, path.Join(*outputImportPath, outputDir))
	for _, rsc := range state.Resources {
		internal.GenResource(rsc, permissionPath, rsc.Name)
	}
}
