package model

// CodeAnchor records explicit code identity at a successful checkpoint.
type CodeAnchor struct {
	Path    string   `json:"path" yaml:"path"`
	SHA256  string   `json:"sha256" yaml:"sha256"`
	Symbols []string `json:"symbols,omitempty" yaml:"symbols,omitempty"`
	Imports []string `json:"imports,omitempty" yaml:"imports,omitempty"`
}
