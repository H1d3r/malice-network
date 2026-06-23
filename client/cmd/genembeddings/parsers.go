package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/chainreactors/malice-network/client/core"
)

type cmdSection struct {
	name, category, source, desc, text string
}

// --- Markdown parser ---

func parseMD(content, source string) []cmdSection {
	var out []cmdSection
	var group, name, desc string
	var buf strings.Builder
	flush := func() {
		if name != "" && buf.Len() > 0 {
			out = append(out, cmdSection{name, group, source, desc, buf.String()})
		}
		name, desc = "", ""
		buf.Reset()
	}
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## ") && !strings.HasPrefix(t, "### ") {
			flush()
			group = strings.TrimPrefix(t, "## ")
		} else if strings.HasPrefix(t, "### ") && !strings.HasPrefix(t, "#### ") {
			flush()
			name = strings.TrimSpace(strings.TrimPrefix(t, "### "))
			if name == "SEE ALSO" {
				name = ""
				continue
			}
			buf.WriteString(name + "\n")
		} else if name != "" {
			if desc == "" && t != "" && !strings.HasPrefix(t, "```") &&
				!strings.HasPrefix(t, "~~~") && !strings.HasPrefix(t, "**") {
				desc = t
			}
			buf.WriteString(line + "\n")
		}
	}
	flush()
	return out
}

// --- Lua parser ---

type luaCmd struct {
	name, description, helpText, source string
}

var (
	reCmd   = regexp.MustCompile(`command\(\s*"([^"]+)"\s*,\s*[^,]+\s*,\s*"([^"]*)"`)
	reOpsec = regexp.MustCompile(`opsec\(\s*"([^"]+)"\s*,\s*([\d.]+)\s*\)`)
	reHelp  = regexp.MustCompile(`help\(\s*"([^"]+)"\s*,\s*\[\[`)
)

func parseLuaDir(dir string) ([]luaCmd, error) {
	var out []luaCmd
	filepath.Walk(dir, func(path string, info os.FileInfo, _ error) error {
		if info == nil || info.IsDir() || filepath.Ext(path) != ".lua" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		out = append(out, parseLua(string(data))...)
		return nil
	})
	return out, nil
}

func parseMalCommunityRepo(dir string) ([]luaCmd, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []luaCmd
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		name := readYAMLName(filepath.Join(dir, e.Name(), "mal.yaml"))
		if name == "" {
			name = e.Name()
		}
		cmds, _ := parseLuaDir(filepath.Join(dir, e.Name()))
		for i := range cmds {
			cmds[i].source = name
		}
		out = append(out, cmds...)
	}
	return out, nil
}

func parseLua(src string) []luaCmd {
	m := make(map[string]*luaCmd)
	for _, match := range reCmd.FindAllStringSubmatch(src, -1) {
		m[match[1]] = &luaCmd{name: match[1], description: match[2]}
	}
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		if match := reHelp.FindStringSubmatch(line); match != nil {
			if cmd, ok := m[match[1]]; ok {
				cmd.helpText = extractHelp(lines, i)
			}
		}
	}
	out := make([]luaCmd, 0, len(m))
	for _, c := range m {
		out = append(out, *c)
	}
	return out
}

func extractHelp(lines []string, start int) string {
	var buf strings.Builder
	started := false
	for i := start; i < len(lines); i++ {
		if !started {
			if idx := strings.Index(lines[i], "[["); idx >= 0 {
				after := lines[i][idx+2:]
				if end := strings.Index(after, "]]"); end >= 0 {
					return strings.TrimSpace(after[:end])
				}
				buf.WriteString(after + "\n")
				started = true
			}
		} else if idx := strings.Index(lines[i], "]]"); idx >= 0 {
			buf.WriteString(lines[i][:idx])
			break
		} else {
			buf.WriteString(lines[i] + "\n")
		}
	}
	s := buf.String()
	if len(s) > 500 {
		s = s[:500]
	}
	return strings.TrimSpace(s)
}

func luaText(c luaCmd) string {
	parts := []string{c.name}
	if c.description != "" {
		parts = append(parts, c.description)
	}
	if c.helpText != "" {
		parts = append(parts, c.helpText)
	}
	return strings.Join(parts, "\n")
}

func readYAMLName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "name:") {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "name:"))
		}
	}
	return ""
}

// BuildCommandText is re-exported for use by genembeddings.
var buildCommandText = core.BuildCommandText

func init() {
	_ = fmt.Sprint // suppress unused
}
