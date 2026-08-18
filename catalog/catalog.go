package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/Moku3956/Project-D/types"
)

// Catalog はテーブルのスキーマ情報をファイルで永続化する。
type Catalog struct {
	mu          sync.RWMutex
	path        string
	schemas     map[string]types.Schema
	nextTableID uint32
}

func NewCatalog(path string) (*Catalog, error) {
	c := &Catalog{
		path:    path,
		schemas: make(map[string]types.Schema),
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return c, nil
	}
	if err := c.load(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Catalog) GetSchema(table string) (*types.Schema, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	s, ok := c.schemas[table]
	if !ok {
		return nil, fmt.Errorf("table %q not found", table)
	}
	return &s, nil
}

func (c *Catalog) TableNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	names := make([]string, 0, len(c.schemas))
	for name := range c.schemas {
		names = append(names, name)
	}
	return names
}

func (c *Catalog) TableExists(table string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.schemas[table]
	return ok
}

func (c *Catalog) CreateTable(schema types.Schema) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.schemas[schema.TableName]; ok {
		return fmt.Errorf("table %q already exists", schema.TableName)
	}
	schema.TableID = c.nextTableID
	c.nextTableID++
	c.schemas[schema.TableName] = schema
	return c.save()
}

func (c *Catalog) DropTable(table string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.schemas[table]; !ok {
		return fmt.Errorf("table %q not found", table)
	}
	delete(c.schemas, table)
	return c.save()
}

// ---- JSON永続化 ----

type columnJSON struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	PrimaryKey bool   `json:"primaryKey"`
	NotNull    bool   `json:"notNull"`
}

type tableJSON struct {
	TableID uint32       `json:"tableID"`
	Columns []columnJSON `json:"columns"`
}

type catalogJSON struct {
	NextTableID uint32               `json:"nextTableID"`
	Tables      map[string]tableJSON `json:"tables"`
}

func (c *Catalog) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}
	var raw catalogJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.nextTableID = raw.NextTableID
	for tableName, t := range raw.Tables {
		cols := make([]types.Column, len(t.Columns))
		for i, col := range t.Columns {
			dt, err := parseDataType(col.Type)
			if err != nil {
				return err
			}
			cols[i] = types.Column{
				Name:       col.Name,
				Type:       dt,
				PrimaryKey: col.PrimaryKey,
				NotNull:    col.NotNull,
			}
		}
		c.schemas[tableName] = types.Schema{
			TableName: tableName,
			TableID:   t.TableID,
			Columns:   cols,
		}
	}
	return nil
}

func (c *Catalog) save() error {
	raw := catalogJSON{
		NextTableID: c.nextTableID,
		Tables:      make(map[string]tableJSON, len(c.schemas)),
	}
	for tableName, s := range c.schemas {
		cols := make([]columnJSON, len(s.Columns))
		for i, col := range s.Columns {
			cols[i] = columnJSON{
				Name:       col.Name,
				Type:       formatDataType(col.Type),
				PrimaryKey: col.PrimaryKey,
				NotNull:    col.NotNull,
			}
		}
		raw.Tables[tableName] = tableJSON{TableID: s.TableID, Columns: cols}
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0644)
}

func parseDataType(s string) (types.DataType, error) {
	switch s {
	case "INT":
		return types.DataType{Kind: types.KindIntType}, nil
	case "BOOL":
		return types.DataType{Kind: types.KindBoolType}, nil
	default:
		var length int
		if _, err := fmt.Sscanf(s, "VARCHAR(%d)", &length); err == nil {
			return types.DataType{Kind: types.KindVarcharType, Length: length}, nil
		}
		return types.DataType{}, fmt.Errorf("unknown type: %s", s)
	}
}

func formatDataType(dt types.DataType) string {
	switch dt.Kind {
	case types.KindIntType:
		return "INT"
	case types.KindBoolType:
		return "BOOL"
	case types.KindVarcharType:
		return fmt.Sprintf("VARCHAR(%d)", dt.Length)
	default:
		return "UNKNOWN"
	}
}
