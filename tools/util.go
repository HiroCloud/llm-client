package tools

import (
	"fmt"
	"github.com/sashabaranov/go-openai/jsonschema"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

// getFunctionName extracts and returns the name of a given function as a string, excluding its package path.
func getFunctionName(i interface{}) string {
	ptr := reflect.ValueOf(i).Pointer()
	funcName := runtime.FuncForPC(ptr).Name()

	// Trim package path if needed
	parts := strings.Split(funcName, "/")
	return parts[len(parts)-1]
}

// getFunctionMetadata extracts function comment and parameter names
func getFunctionMetadata(i interface{}) (string, []string) {
	ptr := reflect.ValueOf(i).Pointer()
	funcName := strings.ReplaceAll(runtime.FuncForPC(ptr).Name(), "-", "_")

	file, _ := runtime.FuncForPC(ptr).FileLine(ptr)
	src, err := os.ReadFile(file)
	if err != nil {
		return "", nil
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file, src, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return "", nil
	}

	var doc string
	var paramNames []string

	// Find function in AST
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			fullFuncName := fmt.Sprintf("%s.%s", node.Name, fn.Name.Name)
			if strings.HasSuffix(funcName, fullFuncName) {

				// Extract function comment
				if fn.Doc != nil {
					doc = strings.TrimSpace(fn.Doc.Text())
				}
				// Extract parameter names
				for _, param := range fn.Type.Params.List {
					for _, name := range param.Names {
						paramNames = append(paramNames, name.Name)
					}
				}
				break
			}
		}
	}

	return doc, paramNames
}

// mapType converts Go types to JSON Schema types
func mapType(t reflect.Type) jsonschema.DataType {
	switch t.Kind() {
	case reflect.String:
		return jsonschema.String
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return jsonschema.Integer
	case reflect.Float32, reflect.Float64:
		return jsonschema.Number
	case reflect.Bool:
		return jsonschema.Boolean
	case reflect.Slice, reflect.Array:
		return jsonschema.Array
	case reflect.Map, reflect.Struct:
		return jsonschema.Object
	default:
		return jsonschema.Null
	}
}

func isCustomType(t reflect.Type) bool {
	return t.Kind() == reflect.String && t.Name() != "" && t.PkgPath() != ""
}

// Helper function to retrieve enum values for custom types (Role, PromptStreamCommand, etc.)
func getEnumValuesForCustomType(t reflect.Type) []string {
	// Here, we're assuming that you define constants in the same package with the custom type name as a prefix.
	// For example, `Role` constants would be `RoleUser`, `RoleAdmin`, etc.

	var constants []string
	//packagePath := t.PkgPath()
	// Get the package path of the type.
	packagePath := t.PkgPath()
	if packagePath == "" {
		return nil
	}

	//path, err := os.Executable()
	//if err != nil {
	//	log.Println(err)
	//}

	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	dir, err := os.ReadDir(joinPaths(cwd, packagePath))
	if err != nil {
		return nil
	}
	for _, d := range dir {
		f := path.Join(joinPaths(cwd, packagePath), d.Name())

		data, err := os.ReadFile(f)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		enumPattern := fmt.Sprintf(`\s+%s([A-Za-z0-9_]+)\s*=\s*%s\(["']([A-Za-z0-9_]+)["']\)`, t.Name(), t.Name())
		m := regexp.MustCompile(enumPattern)
		for _, line := range lines {
			matches := m.FindStringSubmatch(line)
			if len(matches) > 2 {
				// The second capture group will contain the value of the enum (e.g., "user", "assistant")
				constants = append(constants, matches[2])
			}
		}

	}
	return constants
}

// joinPaths joins two paths, ensuring no duplication of base path prefixes.
func joinPaths(basePath, subPath string) string {
	// Ensure basePath doesn't contain the prefix of subPath to avoid duplication
	bps := strings.Split(basePath, "/")
	sps := strings.Split(subPath, "/")
	var ps []string
	for _, p := range bps {
		if p == sps[0] {
			for _, p := range sps {
				ps = append(ps, p)
			}
			return strings.Join(ps, "/")
		} else {
			ps = append(ps, p)
		}
	}
	// Join the paths
	return ""
}
