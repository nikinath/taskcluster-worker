// schema-gen is responsible for auto-generating plugin and engine code based
// on static yml json schema files
package main

import (
	"fmt"
	"go/build"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/docopt/docopt-go"
	"github.com/ghodss/yaml"
	"github.com/taskcluster/jsonschema2go"
)

var (
	version = "schema-gen 1.0.0"
	usage   = `
schema-gen
schema-gen is a tool for generating source code for a plugin or engine. It is
designed to be run from inside a plugin or engine package of a
taskcluster-worker source code directory (package). It then generates go source
code files in the current directory, based on the files it discovers.

If it finds a file called config-schema.yml it will generate a function
ConfigSchema() which returns the content of the given file as json (converted
from yaml) as a string constant. It will also generate go types to represent
the json schema types, so that data conforming to the schema can be
unmarshalled into these types.

Similarly, if it finds a file called payload-schema.yml, it will do the same,
but the generated function will be called PayloadSchema().

Note, since plugins and engines may not require config nor payload data, it is
not necessary for config-schema.yml nor payload-schema.yml to exist.

Please also note, it is recommended to set environment variable GOPATH in order
for schema-gen to correctly determine the correct package name.


  Usage:
    schema-gen
    schema-gen -h|--help
    schema-gen --version

  Options:
    -h --help              Display this help text.
    --version              Display the version (` + version + `).
`
)

type ymlToGoConvertion struct {
	ymlFile    string
	goFunction string
}

func main() {
	// Clear all logging fields, such as timestamps etc...
	log.SetFlags(0)
	log.SetPrefix("schema-gen: ")

	// Parse the docopt string and exit on any error or help message.
	_, err := docopt.Parse(usage, nil, true, version, false, true)
	if err != nil {
		log.Fatalf("ERROR: Cannot parse arguments: %s\n", err)
	}

	conversions := []ymlToGoConvertion{
		ymlToGoConvertion{
			ymlFile:    "config-schema.yml",
			goFunction: "ConfigSchema",
		},
		ymlToGoConvertion{
			ymlFile:    "payload-schema.yml",
			goFunction: "PayloadSchema",
		},
	}

	// Get working directory
	currentFolder, err := os.Getwd()
	if err != nil {
		log.Fatalf("Unable to obtain current working directory: %s", err)
	}

	// Read current package
	pkg, err := build.ImportDir(currentFolder, build.AllowBinary)
	if err != nil {
		log.Fatalf("ERROR: Failed to determine go package inside directory '%s' - is your GOPATH set correctly ('%s')? Error: %s", currentFolder, os.Getenv("GOPATH"), err)
	}

	// Generate go types...
	urls := []string{}
	funcs := []string{}
	for _, c := range conversions {
		ymlFile := filepath.Join(currentFolder, c.ymlFile)
		if _, err := os.Stat(ymlFile); err == nil {
			log.Printf("Found yaml file '%v'", ymlFile)
			urls = append(urls, "file://"+ymlFile)
			funcs = append(funcs, generateSchemaFunction(ymlFile, c.goFunction))
		}
	}
	funcsGoFile := filepath.Join(currentFolder, "generated_functions.go")
	typesGoFile := filepath.Join(currentFolder, "generated_types.go")
	if len(urls) > 0 {
		log.Printf("Generating '%v' ...", typesGoFile)
		generatedCode, _, err := jsonschema2go.Generate(pkg.Name, urls...)
		if err != nil {
			log.Fatalf("ERROR: Problem assembling content for file '%v': %s", typesGoFile, err)
		}
		ioutil.WriteFile(typesGoFile, generatedCode, 0644)
		if err != nil {
			log.Fatalf("ERROR: Could not write generated source code to file '%v': %s", typesGoFile, err)
		}
	}
	if len(funcs) > 0 {
		generatedCode := "package " + pkg.Name + "\n\n"
		for _, f := range funcs {
			generatedCode += f + "\n\n"
		}
		sourceCode, err := format.Source([]byte(generatedCode))
		if err != nil {
			log.Fatalf("ERROR: Could not format generated source code for file '%v': %s", funcsGoFile, err)
		}
		ioutil.WriteFile(funcsGoFile, []byte(sourceCode), 0644)
		if err != nil {
			log.Fatalf("ERROR: Could not write generated source code to file '%v': %s", funcsGoFile, err)
		}
	}
}

func generateSchemaFunction(ymlFile, goFunction string) string {
	data, err := ioutil.ReadFile(ymlFile)
	if err != nil {
		log.Fatalf("ERROR: Problem reading from file '%v' - %s", ymlFile, err)
	}
	// json is valid YAML, so we can safely convert, even if it is already json
	rawJson, err := yaml.YAMLToJSON(data)
	if err != nil {
		log.Fatalf("ERROR: Problem converting file '%v' to json format - %s", ymlFile, err)
	}
	return `func ` + goFunction + `() string {
		return ` + fmt.Sprintf("%s", string(rawJson)) + `
	}`
}