package tools

import (
	"fmt"
	"github.com/sashabaranov/go-openai/jsonschema"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
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

// meta is metadata for a struct
// structComment is the comments above a struct to define what it stores/does
// paramComment stores each of the struct parameter comments
// all comments MUST follow the official tips: https://tip.golang.org/doc/comment
type meta struct {
	structComment string
	paramComment  map[string]string
}

// cleanComment removes comment markers and trims whitespace.
// It also handles the Go convention where the comment often starts
// with the identifier being commented (e.g., "// StructName does...")
func cleanComment(commentText string, identifierName string) string {
	if commentText == "" {
		return ""
	}
	// Remove comment markers
	lines := strings.Split(commentText, "\n")
	cleanedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(line)
		// Remove common Go comment convention prefix if present
		line = strings.TrimPrefix(line, identifierName+" ")
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}

	}
	// Rejoin, preserving potential multi-line structure, but trimmed.
	return strings.TrimSpace(strings.Join(cleanedLines, "\n"))
}

// getStructMetadata retrieves documentation comments for a struct and its fields.
// It parses the source code of the package where the struct is defined.
func getStructMetadata(t reflect.Type) *meta {
	if t == nil || t.Kind() != reflect.Struct {
		log.Println("Input type is nil or not a struct")
		return nil
	}

	// Get the package path and struct name.
	pkgPath := t.PkgPath()
	structName := t.Name()
	if pkgPath == "" {
		// Built-in types or types defined in 'main' package (when run directly)
		// might not have a standard package path accessible this way.
		// AST parsing relies on locating the package source.
		log.Printf("Cannot determine package path for type %s. Is it a built-in type or defined in main?", structName)
		return nil
	}

	// Use go/build to find the package directory.
	// We set UseAllTags to true to potentially match build-constrained files, although
	// type definitions are less likely to be in them.
	buildPkg, err := build.Import(pkgPath, "", build.ImportComment)
	if err != nil {
		log.Printf("Error finding package %s: %v", pkgPath, err)
		return nil
	}
	if buildPkg.Dir == "" {
		log.Printf("Could not find directory for package %s", pkgPath)
		return nil
	}

	// Parse the Go files in the package directory.
	// We need ParseComments mode to include comments in the AST.
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, buildPkg.Dir, nil, parser.ParseComments)
	if err != nil {
		log.Printf("Error parsing package directory %s: %v", buildPkg.Dir, err)
		return nil
	}

	// Find the AST package corresponding to the imported build package.
	// ParseDir returns a map keyed by package name. Use the name from buildPkg.
	astPkg, found := pkgs[buildPkg.Name]
	if !found {
		// This might happen if the package name in the source files differs
		// from the directory name, or if there are multiple packages in the dir.
		// We try using the last part of the import path as a guess.
		keyGuess := pkgPath[strings.LastIndex(pkgPath, "/")+1:]
		astPkg, found = pkgs[keyGuess]
		if !found {
			log.Printf("Could not find AST package '%s' (or guess '%s') in directory %s", buildPkg.Name, keyGuess, buildPkg.Dir)
			return nil
		}
		log.Printf("Found AST package using guess '%s' instead of build name '%s'", keyGuess, buildPkg.Name)
	}

	m := &meta{
		paramComment: make(map[string]string),
	}
	foundStruct := false

	// Iterate through the files in the package AST.
	for _, file := range astPkg.Files {
		// Use ast.Inspect to traverse the AST nodes within the file.
		ast.Inspect(file, func(n ast.Node) bool {
			// Look for type declarations (e.g., "type MyStruct struct { ... }").
			genDecl, ok := n.(*ast.GenDecl)
			// Ensure it's a TYPE declaration. We also need access to genDecl later.
			if !ok || genDecl.Tok != token.TYPE {
				return true // Continue traversal, maybe entering a GenDecl
			}

			// Iterate through the specifications (types) within this GenDecl.
			// Although often just one for standalone types, this loop handles `type ( ... )` blocks too.
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue // Should not happen within a TYPE GenDecl, but safety first
				}

				// Check if this is the struct we're looking for.
				if typeSpec.Name.Name == structName {
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						// Found the type name, but it's not a struct (e.g., type MyInt int)
						//log.Printf("Found type %s, but it is not a struct.", structName)
						return false // Stop searching this branch of the AST
					}

					foundStruct = true // Mark that we found our target struct

					// --- MODIFIED COMMENT EXTRACTION ---
					var structCommentNode *ast.CommentGroup
					if typeSpec.Doc != nil {
						// Primary: Use the doc directly associated with the TypeSpec
						structCommentNode = typeSpec.Doc
						//log.Printf("Struct comment found on TypeSpec.Doc for %s", structName)
					} else if genDecl.Doc != nil {
						// Fallback: Use the doc associated with the parent GenDecl
						// This handles the case shown in the logs where the comment is attached
						// to the 'type Foo struct' statement as a whole.
						structCommentNode = genDecl.Doc
						//		log.Printf("Struct comment found on GenDecl.Doc for %s (TypeSpec.Doc was nil)", structName)
					}

					if structCommentNode != nil {
						m.structComment = cleanComment(structCommentNode.Text(), structName)
					} else {
						// Neither had a comment directly preceding them according to the parser
						//log.Printf("No documentation comment found for struct %s (checked TypeSpec and GenDecl)", structName)
						m.structComment = "" // Ensure it's empty if not found
					}
					// --- END MODIFIED COMMENT EXTRACTION ---

					// Extract comments for each field (this logic remains the same).
					if structType.Fields != nil {
						for _, field := range structType.Fields.List {
							// A field can declare multiple names (e.g., X, Y int).
							for _, name := range field.Names {
								fieldName := name.Name
								fieldComment := ""
								// Prefer doc comment (above the field).
								if field.Doc != nil {
									fieldComment = cleanComment(field.Doc.Text(), fieldName)
								} else if field.Comment != nil {
									// Fallback to line comment (after the field).
									fieldComment = cleanComment(field.Comment.Text(), fieldName)
								}
								if fieldComment != "" {
									m.paramComment[fieldName] = fieldComment
								}
							}
						}
					}
					// We found the struct and processed it, no need to inspect further *within* this GenDecl
					// or continue searching the rest of the file in *this* inspect call.
					// NOTE: Returning false stops traversal down this specific branch (GenDecl -> TypeSpec -> StructType),
					// but the outer loop (`for _, file := range astPkg.Files`) will continue if needed.
					return false
				}
			}
			// If we finished the specs in this GenDecl and didn't find our struct,
			// continue traversing the rest of the file.
			return true
		})

		// If we found the struct in this file, no need to check other files.
		if foundStruct {
			break
		}
	}

	if !foundStruct {
		log.Printf("Struct %s not found in package %s", structName, pkgPath)
		return nil // Struct definition not found in parsed files.
	}
	return m
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
