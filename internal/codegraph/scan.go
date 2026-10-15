package codegraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"knodexa/internal/index"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func Read(root, path string) (model.CodeAnchor, error) {
	var result model.CodeAnchor
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return result, err
	}
	root = canonical
	clean, err := storage.SanitizeRelPath(path)
	if err != nil {
		return result, err
	}
	absolute := filepath.Join(root, clean)
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return result, err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return result, fmt.Errorf("source escapes project: %s", path)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return result, err
	}
	sum := sha256.Sum256(data)
	result = model.CodeAnchor{Path: filepath.ToSlash(clean), SHA256: hex.EncodeToString(sum[:])}
	if filepath.Ext(path) != ".go" {
		return result, nil
	}
	source, err := parser.ParseFile(token.NewFileSet(), path, data, 0)
	if err != nil {
		return result, err
	}
	for _, spec := range source.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return result, err
		}
		result.Imports = append(result.Imports, value)
	}
	for _, decl := range source.Decls {
		switch item := decl.(type) {
		case *ast.FuncDecl:
			name := item.Name.Name
			if item.Recv != nil && len(item.Recv.List) > 0 {
				receiver := item.Recv.List[0].Type
				if star, ok := receiver.(*ast.StarExpr); ok {
					receiver = star.X
				}
				if identifier, ok := receiver.(*ast.Ident); ok {
					name = identifier.Name + "." + name
				}
			}
			result.Symbols = append(result.Symbols, name)
		case *ast.GenDecl:
			for _, spec := range item.Specs {
				if typ, ok := spec.(*ast.TypeSpec); ok {
					result.Symbols = append(result.Symbols, typ.Name.Name)
				}
			}
		}
	}
	sort.Strings(result.Symbols)
	sort.Strings(result.Imports)
	return result, nil
}

// Scan hashes source files but reparses only changed content. The cache is derived,
// never authoritative; malformed current source is skipped until it parses again.
func Scan(root string, db *index.DB) ([]model.CodeAnchor, error) {
	var result []model.CodeAnchor
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".knodexa", "vendor", "node_modules", "audit", "bin":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, _ := filepath.Rel(root, path)
		relative = filepath.ToSlash(relative)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		cached, err := db.GetCodeCache(relative, hash)
		if err != nil {
			return err
		}
		var anchor model.CodeAnchor
		if cached != nil {
			if err := json.Unmarshal(cached, &anchor); err != nil {
				return err
			}
		} else {
			anchor, err = Read(root, relative)
			if err != nil {
				return nil
			}
			encoded, _ := json.Marshal(anchor)
			if err := db.PutCodeCache(relative, hash, encoded); err != nil {
				return err
			}
		}
		result = append(result, anchor)
		return nil
	})
	return result, err
}
