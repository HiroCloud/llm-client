package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Output Collector struct to hold all the analysis results
type AnalysisOutput struct {
	FunctionName       string
	FunctionDefinition string
	Receiver           []string
	Inputs             []string
	Outputs            []string
	StructDefinitions  []string
}

func main() {
	// Parse command line arguments
	filePath := flag.String("file", "", "Path to the Go file to analyze")
	funcName := flag.String("func", "", "Name of the function to find")
	replaceBody := flag.Bool("replacebody", false, "New body for the function to replace the existing one (e.g., 'return nil').")
	flag.Parse()

	if *filePath == "" || *funcName == "" {
		fmt.Println("Usage: go run analyzer.go -file <path> -func <name>")
		// Note: -replacebody is optional for analysis mode
		os.Exit(1)
	}

	// Create the file set
	fset := token.NewFileSet()

	// Parse the entire directory of the provided file to find all package definitions.
	dirPath := filepath.Dir(*filePath)
	pkgs, err := parser.ParseDir(fset, dirPath, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Error parsing directory %s: %v", dirPath, err)
	}

	// Get the main package node (assuming we are interested in the package the file belongs to)
	var mainPkgNode *ast.Package
	for _, pkg := range pkgs {
		// Use the package name of the file
		if _, ok := pkg.Files[filepath.Base(*filePath)]; ok {
			mainPkgNode = pkg
			break
		}
	}

	if mainPkgNode == nil {
		log.Fatalf("Could not find the package containing file: %s", *filePath)
	}

	var targetFunc *ast.FuncDecl

	// 1. Find the function by name in the entire package
	for _, file := range mainPkgNode.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok {
				if fn.Name.Name == *funcName {
					targetFunc = fn
					return false // Found the function
				}
			}
			return true
		})
		if targetFunc != nil {
			break
		}
	}

	if targetFunc == nil {
		fmt.Printf("Function '%s' not found in package in directory %s\n", *funcName, dirPath)
		return
	}

	// --- End Modification Mode Check ---

	// --- Analysis Mode (Original Logic) ---

	// Queue of types to find definitions for
	// key format: "PackageAlias.TypeName" or just "TypeName" for local
	var typeQueue []string
	// Set to keep track of what we've already queued/processed to avoid infinite loops
	processedTypes := make(map[string]bool)

	// Initialize the output struct
	output := AnalysisOutput{
		FunctionName: *funcName,
	}

	// Collect the full function definition
	var funcBuf bytes.Buffer
	if err := format.Node(&funcBuf, fset, targetFunc); err != nil {
		output.FunctionDefinition = fmt.Sprintf("Error formatting function: %v", err)
	} else {
		output.FunctionDefinition = funcBuf.String()
	}

	// Helper to add unique types to queue
	addToQueue := func(types []string) {
		for _, t := range types {
			if !processedTypes[t] {
				processedTypes[t] = true
				typeQueue = append(typeQueue, t)
			}
		}
	}

	// 2. Analyze Receiver (if method)
	if targetFunc.Recv != nil {
		for _, field := range targetFunc.Recv.List {
			types := collectBaseTypes(field.Type, "")
			addToQueue(types)

			typeName := formatTypeExpr(field.Type)
			names := getFieldNames(field.Names)
			output.Receiver = append(output.Receiver, fmt.Sprintf("  - %s: %s", names, typeName))
		}
	}

	// 3. Analyze Inputs
	if targetFunc.Type.Params != nil {
		for _, field := range targetFunc.Type.Params.List {
			// Collect types and add to queue
			types := collectBaseTypes(field.Type, "")
			addToQueue(types)

			typeName := formatTypeExpr(field.Type)
			names := getFieldNames(field.Names)
			output.Inputs = append(output.Inputs, fmt.Sprintf("  - %s: %s", names, typeName))
		}
	} else {
		output.Inputs = append(output.Inputs, "  (None)")
	}

	// 4. Analyze Outputs
	if targetFunc.Type.Results != nil {
		for _, field := range targetFunc.Type.Results.List {
			// Collect types and add to queue
			types := collectBaseTypes(field.Type, "")
			addToQueue(types)

			typeName := formatTypeExpr(field.Type)
			names := getFieldNames(field.Names)
			if names == "" {
				output.Outputs = append(output.Outputs, fmt.Sprintf("  - %s", typeName))
			} else {
				output.Outputs = append(output.Outputs, fmt.Sprintf("  - %s: %s", names, typeName))
			}
		}
	} else {
		output.Outputs = append(output.Outputs, "  (None)")
	}

	// 5. Find and Collect Struct Definitions (Recursive)

	// Process the queue
	structLocations := map[string]struct {
		FilePath string
		Start    int
		End      int
	}{}
	for i := 0; i < len(typeQueue); i++ {
		typeKey := typeQueue[i]

		var structNode *ast.StructType
		var currentAlias string

		if strings.Contains(typeKey, ".") {
			// Handle Imported Type
			parts := strings.SplitN(typeKey, ".", 2)
			pkgAlias, structName := parts[0], parts[1]
			currentAlias = pkgAlias

			// Resolve import path from alias (using the root file's imports)
			importPath, err := resolveImportPath(mainPkgNode.Files[filepath.Base(*filePath)], pkgAlias)
			if err != nil {
				// Often sub-structs in imported packages might trigger this if the alias logic fails
				// or if it's a deep dependency. For now, we skip if we can't resolve.
				continue
			}

			// Locate package on disk
			pkg, err := build.Default.Import(importPath, dirPath, 0)
			if err != nil {
				continue
			}

			// Search in dir
			structNode = findStructInDir(pkg.Dir, structName)

		} else {
			// Handle Local Type
			currentAlias = ""
			var fp string
			var s, e int
			// Now search in the entire parsed package
			structNode, s, e, fp = findStructInLocalPackage(mainPkgNode, typeKey)
			structLocations[typeKey] = struct {
				FilePath string
				Start    int
				End      int
			}{FilePath: fp, Start: s, End: e}
		}

		if structNode != nil {
			// Print the struct to a string and collect it
			displayName := typeKey
			if strings.Contains(displayName, ".") {
				parts := strings.Split(displayName, ".")
				displayName = parts[1]
			}

			var structBuf bytes.Buffer
			err := format.Node(&structBuf, fset, structNode)
			if err != nil {
				output.StructDefinitions = append(output.StructDefinitions, fmt.Sprintf("Error formatting struct %s: %v", displayName, err))
			} else {
				output.StructDefinitions = append(output.StructDefinitions, fmt.Sprintf("type %s %s", displayName, structBuf.String()))
			}

			// RECURSION: Scan fields for more types
			for _, field := range structNode.Fields.List {
				// We pass currentAlias so that if we are in 'mangadex' package,
				// and find type 'AuthorList', it becomes 'mangadex.AuthorList'
				subTypes := collectBaseTypes(field.Type, currentAlias)
				addToQueue(subTypes)
			}
		}
	}

	// 6. Print all collected output at the end
	fmt.Printf("Found Function: %s\n", output.FunctionName)
	fmt.Println(output.FunctionDefinition)
	fmt.Println(stringsRepeat("-", 30))

	if len(output.Receiver) > 0 {
		fmt.Println("Receiver:")
		for _, line := range output.Receiver {
			fmt.Println(line)
		}
		fmt.Println()
	}

	fmt.Println("Inputs:")
	for _, line := range output.Inputs {
		fmt.Println(line)
	}

	fmt.Println("\nOutputs:")
	for _, line := range output.Outputs {
		fmt.Println(line)
	}

	fmt.Printf("\nStruct Definitions Found:\n")
	fmt.Println(stringsRepeat("-", 30))

	if len(output.StructDefinitions) == 0 {
		fmt.Println("No related struct definitions found.")
	} else {
		for _, def := range output.StructDefinitions {
			fmt.Println(def)
			fmt.Println()
		}
	}
}

// Helper to format type expression string for display
func formatTypeExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + formatTypeExpr(t.X)
	case *ast.ArrayType:
		return "[]" + formatTypeExpr(t.Elt)
	case *ast.SelectorExpr:
		return formatTypeExpr(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + formatTypeExpr(t.Key) + "]" + formatTypeExpr(t.Value)
	default:
		return fmt.Sprintf("%T", t)
	}
}

// Helper to get comma separated variable names
func getFieldNames(idents []*ast.Ident) string {
	if len(idents) == 0 {
		return ""
	}
	var names []string
	for _, id := range idents {
		names = append(names, id.Name)
	}
	return joinStrings(names, ", ")
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	res := strs[0]
	for i := 1; i < len(strs); i++ {
		res += sep + strs[i]
	}
	return res
}

func stringsRepeat(s string, count int) string {
	res := ""
	for i := 0; i < count; i++ {
		res += s
	}
	return res
}

func isPrimitive(t string) bool {
	switch t {
	case "bool", "uint", "uint8", "uint16", "uint32", "uint64",
		"int", "int8", "int16", "int32", "int64",
		"float32", "float64", "complex64", "complex128",
		"string", "byte", "rune", "uintptr", "error":
		return true
	}
	return false
}

func collectBaseTypes(expr ast.Expr, prefix string) []string {
	var types []string

	switch t := expr.(type) {

	case *ast.Ident:
		if !isPrimitive(t.Name) {
			if prefix != "" {
				types = append(types, prefix+"."+t.Name)
			} else {
				types = append(types, t.Name)
			}
		}

	case *ast.StarExpr:
		return collectBaseTypes(t.X, prefix)

	case *ast.ArrayType:
		return collectBaseTypes(t.Elt, prefix)

	case *ast.MapType:
		types = append(types, collectBaseTypes(t.Key, prefix)...)
		types = append(types, collectBaseTypes(t.Value, prefix)...)

	case *ast.SelectorExpr:
		// pkg.Type
		if ident, ok := t.X.(*ast.Ident); ok {
			pkg := ident.Name
			name := t.Sel.Name
			types = append(types, pkg+"."+name)
		}

	case *ast.StructType:
		// anonymous struct, recurse fields
		for _, f := range t.Fields.List {
			types = append(types, collectBaseTypes(f.Type, prefix)...)
		}

	}

	return types
}

func resolveImportPath(file *ast.File, alias string) (string, error) {

	for _, imp := range file.Imports {

		path := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil {
			if imp.Name.Name == alias {
				return path, nil
			}
		} else {
			parts := strings.Split(path, "/")
			if parts[len(parts)-1] == alias {
				return path, nil
			}
		}

	}

	return "", fmt.Errorf("import alias %s not found", alias)
}

func findStructInLocalPackage(pkg *ast.Package, name string) (*ast.StructType, int, int, string) {

	for filePath, file := range pkg.Files {

		for _, decl := range file.Decls {

			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}

			for _, spec := range gen.Specs {

				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if ts.Name.Name != name {
					continue
				}

				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}

				start := file.Pos()
				end := file.End()

				return st,
					int(start),
					int(end),
					filePath
			}
		}
	}

	return nil, 0, 0, ""
}

func findStructInDir(dir, structName string) *ast.StructType {

	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return nil
	}

	for _, pkg := range pkgs {

		for _, file := range pkg.Files {

			for _, decl := range file.Decls {

				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}

				for _, spec := range gen.Specs {

					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					if ts.Name.Name != structName {
						continue
					}

					st, ok := ts.Type.(*ast.StructType)
					if ok {
						return st
					}

				}
			}
		}
	}

	return nil
}
