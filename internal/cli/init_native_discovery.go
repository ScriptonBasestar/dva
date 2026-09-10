package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ScriptonBasestar/dva/internal/config"
)

var portMappingsManifestNames = []string{
	"PORT_MAPPINGS.yaml",
	"PORT_MAPPINGS.yml",
	"port_mappings.yaml",
	"port_mappings.yml",
}

type portMapping struct {
	Name        string
	Category    string
	Port        int
	Env         string
	Description string
}

type makeTarget struct {
	Name        string
	Description string
	Recipe      []string
	Unsupported string
}

type nativeScaffoldEntry struct {
	Name        string
	Description string
	Dir         string
	Run         string
	Build       string
	Port        *portMapping
}

type nativeScaffoldDiscovery struct {
	entries     []nativeScaffoldEntry
	subprojects map[string]string
	ports       []portMapping
	portSource  string
	hasTest     bool
}

func (d nativeScaffoldDiscovery) hasEvidence() bool {
	return len(d.entries) > 0 || len(d.subprojects) > 0 || len(d.ports) > 0
}

// parsePortMappingsManifest reads the first manifest in the documented,
// case-sensitive precedence order. A second spelling cannot silently merge or
// override the first one.
func parsePortMappingsManifest(dir string) ([]portMapping, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", dir, err)
	}
	present := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			present[entry.Name()] = true
		}
	}
	for _, name := range portMappingsManifestNames {
		// Check the directory entry spelling before opening. On a
		// case-insensitive filesystem os.ReadFile(PORT_MAPPINGS.yaml) would
		// otherwise also open port_mappings.yaml and falsify precedence.
		if !present[name] {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("read %s: %w", path, err)
		}

		var document yaml.Node
		if err := yaml.Unmarshal(data, &document); err != nil {
			return nil, "", fmt.Errorf("parse %s: %w", path, err)
		}
		root := yamlDocumentRoot(&document)
		services := yamlMappingValue(root, "services")
		var mappings []portMapping
		collectPortMappings(services, "", &mappings)
		sort.Slice(mappings, func(i, j int) bool {
			if mappings[i].Category != mappings[j].Category {
				return mappings[i].Category < mappings[j].Category
			}
			return mappings[i].Name < mappings[j].Name
		})
		return mappings, name, nil
	}
	return nil, "", nil
}

func yamlDocumentRoot(document *yaml.Node) *yaml.Node {
	if document != nil && document.Kind == yaml.DocumentNode && len(document.Content) == 1 {
		return document.Content[0]
	}
	return document
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func collectPortMappings(node *yaml.Node, category string, result *[]portMapping) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		name, value := node.Content[i].Value, node.Content[i+1]
		portNode := yamlMappingValue(value, "port")
		if portNode != nil {
			var port int
			if err := portNode.Decode(&port); err != nil || port < 1 || port > 65535 {
				continue
			}
			mapping := portMapping{Name: name, Category: category, Port: port}
			if env := yamlMappingValue(value, "env"); env != nil {
				mapping.Env = strings.TrimSpace(env.Value)
			}
			if description := yamlMappingValue(value, "description"); description != nil {
				mapping.Description = strings.TrimSpace(description.Value)
			}
			*result = append(*result, mapping)
			continue
		}
		nextCategory := category
		if nextCategory == "" {
			nextCategory = name
		}
		collectPortMappings(value, nextCategory, result)
	}
}

func parseGzGitWorkspaces(dir string) (map[string]string, error) {
	path := filepath.Join(dir, ".gz-git.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var manifest struct {
		Workspaces map[string]struct {
			TargetPath string `yaml:"targetPath"`
			Path       string `yaml:"path"`
		} `yaml:"workspaces"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	result := make(map[string]string, len(manifest.Workspaces))
	for name, workspace := range manifest.Workspaces {
		path := strings.TrimSpace(workspace.TargetPath)
		if path == "" {
			path = strings.TrimSpace(workspace.Path)
		}
		if path == "" {
			path = name
		}
		path = filepath.Clean(path)
		if path == "." || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			continue
		}
		result[name] = filepath.ToSlash(path)
	}
	return result, nil
}

func parseMakefileTargets(dir string) ([]makeTarget, error) {
	for _, name := range []string{"Makefile", "GNUmakefile", "makefile"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		return parseMakefileTargetData(string(data)), nil
	}
	return nil, nil
}

func parseMakefileTargetData(data string) []makeTarget {
	lines := strings.Split(data, "\n")
	variables := parseSimpleMakeVariables(lines)
	var targets []makeTarget
	for i := range lines {
		line := lines[i]
		if line == "" || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") ||
			strings.HasPrefix(line, "#") || strings.HasPrefix(line, ".") {
			continue
		}
		left, right, ok := strings.Cut(line, ":")
		if !ok || strings.Contains(left, "=") || strings.Contains(left, "%") || strings.Contains(left, "$") {
			continue
		}
		description := ""
		if _, comment, found := strings.Cut(right, "##"); found {
			description = strings.TrimSpace(comment)
		}
		var recipe []string
		unsupported := ""
		for j := i + 1; j < len(lines) && strings.HasPrefix(lines[j], "\t"); j++ {
			command := strings.TrimLeft(strings.TrimSpace(lines[j]), "@-+")
			command = strings.TrimSpace(command)
			if command == "" || strings.HasPrefix(command, "#") {
				continue
			}
			expanded := expandMakeVariables(command, variables, map[string]bool{})
			normalized, unsupportedReference := normalizeMakeRecipeDollars(expanded)
			if unsupportedReference != "" {
				if unsupported == "" {
					unsupported = unsupportedReference
				}
			} else {
				command = normalized
			}
			recipe = append(recipe, command)
		}
		for name := range strings.FieldsSeq(left) {
			if name == "" || strings.HasPrefix(name, ".") {
				continue
			}
			targets = append(targets, makeTarget{Name: name, Description: description, Recipe: recipe, Unsupported: unsupported})
		}
	}
	sort.SliceStable(targets, func(i, j int) bool { return targets[i].Name < targets[j].Name })
	return targets
}

func splitRecipeWorkingDir(recipe []string) (dir, command string, ok bool) {
	if len(recipe) == 0 {
		return "", "", false
	}
	commands := append([]string(nil), recipe...)
	firstDir := ""
	sawWorkingDir := false
	sawRootCommand := false
	for i, line := range commands {
		before, after, found := strings.Cut(line, "&&")
		before = strings.TrimSpace(before)
		if !found || !strings.HasPrefix(before, "cd ") {
			sawRootCommand = true
			continue
		}
		sawWorkingDir = true
		candidate := strings.TrimSpace(strings.TrimPrefix(before, "cd "))
		candidate = strings.Trim(candidate, `"'`)
		if candidate == "" || filepath.IsAbs(candidate) || candidate == ".." || strings.HasPrefix(candidate, "../") {
			return "", "", false
		}
		if firstDir == "" {
			firstDir = filepath.ToSlash(filepath.Clean(candidate))
		}
		if filepath.ToSlash(filepath.Clean(candidate)) != firstDir {
			return "", "", false
		}
		commands[i] = strings.TrimSpace(after)
	}
	if sawWorkingDir && sawRootCommand {
		return "", "", false
	}
	return firstDir, strings.Join(commands, " && "), true
}

func discoverNativeScaffold(dir string) (nativeScaffoldDiscovery, error) {
	ports, portSource, err := parsePortMappingsManifest(dir)
	if err != nil {
		return nativeScaffoldDiscovery{}, err
	}
	subprojects, err := parseGzGitWorkspaces(dir)
	if err != nil {
		return nativeScaffoldDiscovery{}, err
	}
	if subprojects == nil {
		subprojects = map[string]string{}
	}
	targets, err := parseMakefileTargets(dir)
	if err != nil {
		return nativeScaffoldDiscovery{}, err
	}
	targetByName := make(map[string]makeTarget, len(targets))
	for _, target := range targets {
		targetByName[target.Name] = target
	}

	result := nativeScaffoldDiscovery{subprojects: subprojects, ports: ports, portSource: portSource}
	if testTarget, exists := targetByName["test"]; exists && len(testTarget.Recipe) > 0 {
		if testTarget.Unsupported != "" {
			return nativeScaffoldDiscovery{}, fmt.Errorf("makefile target test uses unsupported or unresolved variable %s", testTarget.Unsupported)
		}
		result.hasTest = true
	}
	for _, target := range targets {
		name, found := strings.CutPrefix(target.Name, "dev-")
		if !found || name == "" {
			continue
		}
		if target.Unsupported != "" {
			return nativeScaffoldDiscovery{}, fmt.Errorf("makefile target %s uses unsupported or unresolved variable %s", target.Name, target.Unsupported)
		}
		dir, run, ok := splitRecipeWorkingDir(target.Recipe)
		if !ok || run == "" || isInstructionOnlyRecipe(target.Recipe) {
			continue
		}
		entry := nativeScaffoldEntry{Name: name, Description: target.Description, Dir: dir, Run: run}
		if build, exists := targetByName["build-"+name]; exists {
			if build.Unsupported != "" {
				return nativeScaffoldDiscovery{}, fmt.Errorf("makefile target %s uses unsupported or unresolved variable %s", build.Name, build.Unsupported)
			}
			buildDir, buildCommand, valid := splitRecipeWorkingDir(build.Recipe)
			if valid && buildCommand != "" && buildDir == dir {
				entry.Build = buildCommand
			}
		}
		if entry.Build == "" {
			if build := targetByName["build"]; build.Unsupported != "" {
				return nativeScaffoldDiscovery{}, fmt.Errorf("makefile target build uses unsupported or unresolved variable %s", build.Unsupported)
			}
			entry.Build = buildCommandForDir(targetByName["build"].Recipe, dir)
		}
		for i := range ports {
			if ports[i].Name == name && (ports[i].Category == "" || ports[i].Category == "application") {
				entry.Port = &ports[i]
				break
			}
		}
		result.entries = append(result.entries, entry)
	}
	return result, nil
}

func buildCommandForDir(recipe []string, dir string) string {
	if dir == "" {
		return ""
	}
	var commands []string
	for _, line := range recipe {
		buildDir, command, ok := splitRecipeWorkingDir([]string{line})
		if ok && buildDir == dir && command != "" {
			commands = append(commands, command)
		}
	}
	return strings.Join(commands, " && ")
}

func isInstructionOnlyRecipe(recipe []string) bool {
	if len(recipe) == 0 {
		return false
	}
	for _, command := range recipe {
		if !strings.HasPrefix(strings.TrimSpace(command), "echo ") && !strings.HasPrefix(strings.TrimSpace(command), "printf ") {
			return false
		}
	}
	return true
}

func generateDiscoveredConfig(discovery nativeScaffoldDiscovery, lang string, evidence langEvidence) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "version: %s\n", strconv.Quote(config.MinScaffoldVersion))
	if lang != "" {
		_, _ = fmt.Fprintf(&b, "\n# A %s was detected", evidence.phrase(lang))
		if len(discovery.entries) == 0 {
			b.WriteString(", but no declared dev-* recipe was found")
		}
		b.WriteString(".\n")
	}
	if len(discovery.entries) > 0 {
		b.WriteString("\nstack:\n")
		for _, entry := range discovery.entries {
			_, _ = fmt.Fprintf(&b, "  %s:\n", strconv.Quote(entry.Name))
			if entry.Description != "" {
				_, _ = fmt.Fprintf(&b, "    description: %s\n", strconv.Quote(entry.Description))
			}
			b.WriteString("    default_runner: native\n    runners:\n      native:\n")
			if entry.Dir != "" {
				_, _ = fmt.Fprintf(&b, "        dir: %s\n", strconv.Quote(entry.Dir))
			}
			if entry.Build != "" {
				_, _ = fmt.Fprintf(&b, "        build: %s\n", strconv.Quote(entry.Build))
			}
			_, _ = fmt.Fprintf(&b, "        run: %s\n", strconv.Quote(entry.Run))
			if entry.Port != nil {
				_, _ = fmt.Fprintf(&b, "    # %s declares port %d", discovery.portSource, entry.Port.Port)
				if entry.Port.Env != "" {
					_, _ = fmt.Fprintf(&b, " via %s", entry.Port.Env)
				}
				b.WriteString("; confirm the application consumes it.\n")
				_, _ = fmt.Fprintf(&b, "    health_checks:\n      %s:\n        type: http\n        url: %s\n        ready_timeout: 60\n", strconv.Quote(entry.Name), strconv.Quote(fmt.Sprintf("http://localhost:%d/", entry.Port.Port)))
			}
		}
		b.WriteString("\nplans:\n  dev:\n    description: \"Run discovered Makefile development targets\"\n    entries:\n")
		for i, entry := range discovery.entries {
			_, _ = fmt.Fprintf(&b, "      - name: %s\n        runner: native\n        order: %d\n", strconv.Quote(entry.Name), (i+1)*10)
		}
		b.WriteString("default_plan: dev\n")
	}
	if discovery.hasTest {
		b.WriteString("\ninteraction:\n  test:\n    description: \"Run the declared Makefile test target\"\n    runner: local\n    command: \"make test\"\n")
	}
	if len(discovery.subprojects) > 0 {
		b.WriteString("\nsubprojects:\n")
		names := make([]string, 0, len(discovery.subprojects))
		for name := range discovery.subprojects {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			_, _ = fmt.Fprintf(&b, "  %s:\n    path: %s\n", strconv.Quote(name), strconv.Quote(discovery.subprojects[name]))
		}
	}
	endpointMappings := map[string]portMapping{}
	for _, mapping := range discovery.ports {
		if mapping.Category == "" || mapping.Category == "application" {
			if _, exists := endpointMappings[mapping.Name]; !exists {
				endpointMappings[mapping.Name] = mapping
			}
		}
	}
	if len(endpointMappings) > 0 {
		b.WriteString("\nendpoints:\n")
		names := make([]string, 0, len(endpointMappings))
		for name := range endpointMappings {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			mapping := endpointMappings[name]
			_, _ = fmt.Fprintf(&b, "  %s:\n    url: %s\n", strconv.Quote(name), strconv.Quote(fmt.Sprintf("http://localhost:%d", mapping.Port)))
			label := mapping.Description
			if label == "" {
				label = name
			}
			_, _ = fmt.Fprintf(&b, "    label: %s\n", strconv.Quote(label))
		}
	}
	return b.String()
}
