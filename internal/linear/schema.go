package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const schemaFileName = "schema.json"

type schemaCache struct {
	FetchedAt   time.Time                 `json:"fetched_at"`
	QueryType   schemaTypeInfo            `json:"query"`
	IssueType   schemaTypeInfo            `json:"issue"`
	CommentType schemaTypeInfo            `json:"comment"`
	UserType    schemaTypeInfo            `json:"user"`
	Types       map[string]schemaTypeInfo `json:"types,omitempty"`
}

type schemaTypeInfo struct {
	Fields []schemaField `json:"fields"`
}

type schemaField struct {
	Name string        `json:"name"`
	Args []schemaArg   `json:"args"`
	Type schemaGQLType `json:"type"`
}

type schemaArg struct {
	Name string        `json:"name"`
	Type schemaGQLType `json:"type"`
}

type schemaGQLType struct {
	Kind   string         `json:"kind"`
	Name   string         `json:"name"`
	OfType *schemaGQLType `json:"ofType"`
}

func (t schemaGQLType) baseName() string {
	cur := &t
	for cur != nil {
		if cur.Name != "" {
			return cur.Name
		}
		cur = cur.OfType
	}
	return ""
}

func DefaultSchemaPath() (string, error) {
	if base := os.Getenv("XDG_DATA_HOME"); base != "" {
		return filepath.Join(base, "linear", schemaFileName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".local", "share", "linear", schemaFileName), nil
}

func (c *Client) loadSchema(ctx context.Context) (*schemaCache, error) {
	if c.schemaPath == "" {
		return c.fetchSchema(ctx)
	}

	schema, ok := loadSchemaCache(c.schemaPath)
	if ok {
		if time.Since(schema.FetchedAt) < 24*time.Hour {
			return schema, nil
		}
		fresh, err := c.fetchSchema(ctx)
		if err == nil {
			_ = saveSchemaCache(c.schemaPath, fresh)
			return fresh, nil
		}
		return schema, nil
	}

	fresh, err := c.fetchSchema(ctx)
	if err != nil {
		return nil, err
	}
	_ = saveSchemaCache(c.schemaPath, fresh)
	return fresh, nil
}

func (c *Client) loadSchemaType(ctx context.Context, name string) (schemaTypeInfo, bool, error) {
	cache, err := c.loadSchema(ctx)
	if err != nil {
		return schemaTypeInfo{}, false, err
	}
	if cache.Types != nil {
		if info, ok := cache.Types[name]; ok && len(info.Fields) > 0 {
			return info, true, nil
		}
	}

	query := `query($name: String!) {
  __type(name: $name) {
    fields { name type { kind name ofType { kind name ofType { kind name ofType { kind name } } } } }
  }
}`
	var resp struct {
		Type *struct {
			Fields []schemaField `json:"fields"`
		} `json:"__type"`
	}
	if err := c.do(ctx, query, map[string]any{"name": name}, &resp); err != nil {
		return schemaTypeInfo{}, false, err
	}
	if resp.Type == nil {
		return schemaTypeInfo{}, false, nil
	}
	info := schemaTypeInfo{Fields: resp.Type.Fields}
	if cache.Types == nil {
		cache.Types = map[string]schemaTypeInfo{}
	}
	cache.Types[name] = info
	_ = saveSchemaCache(c.schemaPath, cache)
	return info, true, nil
}

func loadSchemaCache(path string) (*schemaCache, bool) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()

	var cache schemaCache
	if err := json.NewDecoder(file).Decode(&cache); err != nil {
		return nil, false
	}
	if cache.FetchedAt.IsZero() {
		return nil, false
	}
	if cache.Types == nil {
		cache.Types = map[string]schemaTypeInfo{}
	}
	return &cache, true
}

func saveSchemaCache(path string, cache *schemaCache) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp := path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cache); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (c *Client) fetchSchema(ctx context.Context) (*schemaCache, error) {
	query := `query {
  __type(name: "Query") {
    fields {
      name
      args {
        name
        type { kind name ofType { kind name ofType { kind name ofType { kind name } } } }
      }
      type { kind name ofType { kind name ofType { kind name } } }
    }
  }
  issue: __type(name: "Issue") {
    fields { name type { kind name ofType { kind name ofType { kind name } } } }
  }
  comment: __type(name: "Comment") {
    fields { name type { kind name ofType { kind name ofType { kind name } } } }
  }
  user: __type(name: "User") {
    fields { name type { kind name ofType { kind name ofType { kind name } } } }
  }
}`
	var resp struct {
		Type *struct {
			Fields []schemaField `json:"fields"`
		} `json:"__type"`
		Issue *struct {
			Fields []schemaField `json:"fields"`
		} `json:"issue"`
		Comment *struct {
			Fields []schemaField `json:"fields"`
		} `json:"comment"`
		User *struct {
			Fields []schemaField `json:"fields"`
		} `json:"user"`
	}
	if err := c.do(ctx, query, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Type == nil {
		return nil, fmt.Errorf("schema query type not found")
	}
	cache := &schemaCache{
		FetchedAt: time.Now(),
		QueryType: schemaTypeInfo{Fields: resp.Type.Fields},
		Types:     map[string]schemaTypeInfo{},
	}
	if resp.Issue != nil {
		cache.IssueType = schemaTypeInfo{Fields: resp.Issue.Fields}
	}
	if resp.Comment != nil {
		cache.CommentType = schemaTypeInfo{Fields: resp.Comment.Fields}
	}
	if resp.User != nil {
		cache.UserType = schemaTypeInfo{Fields: resp.User.Fields}
	}
	return cache, nil
}

func (s *schemaCache) argBaseType(fieldName, argName string) (string, bool) {
	for _, field := range s.QueryType.Fields {
		if strings.EqualFold(field.Name, fieldName) {
			for _, arg := range field.Args {
				if strings.EqualFold(arg.Name, argName) {
					name := arg.Type.baseName()
					if name != "" {
						return name, true
					}
				}
			}
		}
	}
	return "", false
}

func (s *schemaCache) hasField(typeName, fieldName string) bool {
	var fields []schemaField
	switch strings.ToLower(typeName) {
	case "query":
		fields = s.QueryType.Fields
	case "issue":
		fields = s.IssueType.Fields
	case "comment":
		fields = s.CommentType.Fields
	case "user":
		fields = s.UserType.Fields
	default:
		if s.Types != nil {
			if info, ok := s.Types[typeName]; ok {
				fields = info.Fields
			}
		}
	}
	for _, field := range fields {
		if strings.EqualFold(field.Name, fieldName) {
			return true
		}
	}
	return false
}

func (s *schemaCache) field(typeName, fieldName string) (schemaField, bool) {
	var fields []schemaField
	switch strings.ToLower(typeName) {
	case "query":
		fields = s.QueryType.Fields
	case "issue":
		fields = s.IssueType.Fields
	case "comment":
		fields = s.CommentType.Fields
	case "user":
		fields = s.UserType.Fields
	default:
		if s.Types != nil {
			if info, ok := s.Types[typeName]; ok {
				fields = info.Fields
			}
		}
	}
	for _, field := range fields {
		if strings.EqualFold(field.Name, fieldName) {
			return field, true
		}
	}
	return schemaField{}, false
}
