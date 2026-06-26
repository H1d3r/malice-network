package core

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// SearchIndex provides FTS5 keyword search over the live cobra command tree.
// Complements VectorIndex (semantic) with exact/prefix matching.
type SearchIndex struct {
	mu sync.RWMutex
	db *sql.DB
}

func NewSearchIndex(dbPath string) (*SearchIndex, error) {
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open search db: %w", err)
	}
	si := &SearchIndex{db: db}
	if _, err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
		name, type, category, source, short_desc, long_desc, usage, example, flags,
		ttp UNINDEXED, opsec UNINDEXED, subcommands,
		tokenize='unicode61 remove_diacritics 2'
	)`); err != nil {
		db.Close()
		return nil, err
	}
	return si, nil
}

// Rebuild re-indexes all commands from the given menu sources.
func (si *SearchIndex) Rebuild(sources ...func() []*cobra.Command) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	si.db.Exec("DROP TABLE IF EXISTS search_index")
	si.db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
		name, type, category, source, short_desc, long_desc, usage, example, flags,
		ttp UNINDEXED, opsec UNINDEXED, subcommands,
		tokenize='unicode61 remove_diacritics 2'
	)`)

	tx, err := si.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO search_index(name,type,category,source,short_desc,long_desc,usage,example,flags,ttp,opsec,subcommands) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, src := range sources {
		if src == nil {
			continue
		}
		for _, cmd := range src() {
			indexTree(stmt, cmd)
		}
	}
	return tx.Commit()
}

func indexTree(stmt *sql.Stmt, cmd *cobra.Command) {
	if cmd.Hidden {
		return
	}
	source := cmd.Annotations["source"]
	if source == "" {
		source = "builtin"
	}
	cmdType := "command"
	if source != "builtin" {
		cmdType = "plugin"
	}

	var flags []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) { flags = append(flags, f.Name+" "+f.Usage) })

	var subs []string
	for _, sub := range cmd.Commands() {
		if !sub.Hidden {
			subs = append(subs, sub.Name())
		}
	}

	stmt.Exec(cmd.Name(), cmdType, cmd.GroupID, source, cmd.Short, cmd.Long,
		cmd.UseLine(), cmd.Example, strings.Join(flags, " "),
		cmd.Annotations["ttp"], cmd.Annotations["opsec"], strings.Join(subs, " "))

	for _, sub := range cmd.Commands() {
		indexTree(stmt, sub)
	}
}

// SearchResult holds a single FTS5 search hit.
type SearchResult struct {
	Name        string
	Type        string
	Category    string
	Source      string
	Description string
	Usage       string
	TTP         string
	Opsec       string
	Subcommands string
	Snippet     string
	Rank        float64
}

// Search performs FTS5 full-text search.
func (si *SearchIndex) Search(query, typeFilter, category string, limit int) ([]SearchResult, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	var conds []string
	var args []interface{}
	conds = append(conds, "search_index MATCH ?")
	args = append(args, ftsQuery)
	if typeFilter != "" {
		conds = append(conds, "type = ?")
		args = append(args, typeFilter)
	}
	if category != "" {
		conds = append(conds, "category = ?")
		args = append(args, category)
	}
	args = append(args, limit)

	rows, err := si.db.Query(fmt.Sprintf(
		`SELECT name,type,category,source,short_desc,usage,ttp,opsec,subcommands,
		snippet(search_index,4,'**','**','...',15),rank
		FROM search_index WHERE %s ORDER BY rank LIMIT ?`,
		strings.Join(conds, " AND ")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		rows.Scan(&r.Name, &r.Type, &r.Category, &r.Source, &r.Description,
			&r.Usage, &r.TTP, &r.Opsec, &r.Subcommands, &r.Snippet, &r.Rank)
		results = append(results, r)
	}
	return results, rows.Err()
}

func (si *SearchIndex) Categories(typeFilter string) ([]string, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()

	query := "SELECT DISTINCT category FROM search_index WHERE category != ''"
	args := []interface{}{}
	if typeFilter != "" {
		query += " AND type = ?"
		args = append(args, typeFilter)
	}
	query += " ORDER BY category"

	rows, err := si.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func buildFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	for _, op := range []string{" AND ", " OR ", " NOT ", "\"", "*"} {
		if strings.Contains(query, op) {
			return query
		}
	}
	words := strings.Fields(query)
	var parts []string
	for _, w := range words {
		w = strings.NewReplacer("\"", "", "(", "", ")", "").Replace(w)
		if w != "" {
			parts = append(parts, "\""+w+"\"")
		}
	}
	if len(parts) == 1 {
		return parts[0] + "*"
	}
	return strings.Join(parts, " AND ")
}

func (si *SearchIndex) Close() error {
	if si == nil || si.db == nil {
		return nil
	}
	return si.db.Close()
}
