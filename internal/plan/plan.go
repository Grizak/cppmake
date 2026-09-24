package plan

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Grizak/cppmake/internal/parser"
)

// Rule is a reusable command template, keyed by name. Multiple edges
// can share a rule; only the variables differ per edge.
type Rule struct {
	Name    string
	Command string // uses $in, $out, and any custom vars like $flags
	Depfile string // e.g. "$out.d" — empty if this rule has no deps
	Deps    string // "gcc" for gcc-style depfiles; empty otherwise
}

// Edge is one build step: some inputs produce some outputs via a rule.
type Edge struct {
	Rule string

	Outputs []string
	Inputs  []string // explicit inputs — order matters (e.g. link order)

	// Implicit inputs affect staleness but aren't part of $in.
	// This is how depends_on becomes "rebuild if the dependency changed"
	// without polluting the compiler's argument list.
	ImplicitInputs []string

	// Order-only inputs must exist before this edge runs, but changes
	// to them don't make this edge stale. Good for "create output dir".
	OrderOnlyInputs []string

	Vars map[string]string // rendered into $flags, $libs, etc.
}

// Target carries metadata a backend might want beyond the raw edges —
// e.g. to build "the ctest target" by name, or to know what to add
// to `default`.
type Target struct {
	Name      string
	Type      string // binary, static, shared
	Output    string // final artifact path
	Edges     []Edge // this target's own edges, in dependency order
	DependsOn []string
}

type Plan struct {
	Rules     []Rule
	Targets   []Target
	Defaults  []string // outputs to build when no target is named
	Name      string
	Version   string
	HasStatic bool // True if there is a static build somewhere
}

func Resolve(cfg *parser.Config) (*Plan, error) {
	cfg.ApplyDefaults()
	if errs := Validate(cfg); len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	order, err := topoSort(cfg.Targets) // by depends_on
	if err != nil {
		return nil, err
	}

	p := &Plan{}
	ruleSeen := map[string]bool{}
	outputOf := map[string]string{} // target name -> its output path

	// Name and Version
	p.Name = cfg.Project.Name
	p.Version = cfg.Project.Version

	p.HasStatic = false

	pts := make([]Target, len(order))
	compileRules := make(map[string]string, len(order))
	tcs := make(map[string]parser.Toolchain, len(order))

	for i, t := range order {
		tc, ok := cfg.Toolchains[t.Lang][t.Toolchain]
		if !ok {
			return nil, fmt.Errorf("target %s: unknown toolchain %s.%s", t.Name, t.Lang, t.Toolchain)
		}

		compileRule := fmt.Sprintf("%s_%s", t.Lang, t.Toolchain)
		if !ruleSeen[compileRule] {
			p.Rules = append(p.Rules, buildCompileRule(compileRule, tc))
			ruleSeen[compileRule] = true
		}

		if t.Type == "static" {
			p.HasStatic = true
		}

		pts[i] = Target{Name: t.Name, Type: t.Type}
		compileRules[t.Name] = compileRule
		tcs[t.Name] = tc
	}

	if p.HasStatic {
		p.Rules = append(p.Rules, Rule{
			Name:    "link_ar",
			Command: "ar rcs $out $flags $in $libs",
		})
		ruleSeen["link_ar"] = true
	}

	for i, t := range order {
		pt := pts[i]
		compileRule := compileRules[t.Name]
		tc := tcs[t.Name]
		var objs []string

		for _, src := range t.Src {
			obj := objectPath(cfg.Build.BuildDir, t.Name, src)
			objs = append(objs, obj)
			pt.Edges = append(pt.Edges, Edge{
				Rule:    compileRule,
				Outputs: []string{obj},
				Inputs:  []string{src},
				Vars:    map[string]string{"flags": strings.Join(append(tc.Flags, t.Flags...), " ")},
			})
		}

		// depends_on: pull in each dependency's output as an implicit
		// input, and, for link steps, as an extra input to link against.
		var depOutputs []string
		for _, dep := range t.DependsOn {
			depOutputs = append(depOutputs, outputOf[dep])
		}

		linkRule := linkRuleFor(t, cfg)
		if !ruleSeen[linkRule.Name] {
			p.Rules = append(p.Rules, linkRule)
			ruleSeen[linkRule.Name] = true
		}

		out := outputPath(cfg.Build.BuildDir, t)
		pt.Output = out
		pt.Edges = append(pt.Edges, Edge{
			Rule:    linkRule.Name,
			Outputs: []string{out},
			Inputs:  append(objs, depOutputs...), // link deps' .a files directly
			Vars:    map[string]string{"flags": strings.Join(t.Flags, " "), "libs": strings.Join(t.Libs, " ")},
		})

		outputOf[t.Name] = out
		p.Targets = append(p.Targets, pt)
		p.Defaults = append(p.Defaults, out)
	}

	return p, nil
}

func Validate(cfg *parser.Config) []error {
	var validationErrors []error
	knownTypes := map[string]bool{
		"binary": true,
		"static": true,
		"shared": true,
	}
	targetIndexes := make(map[string]int, len(cfg.Targets))

	for index, target := range cfg.Targets {
		if previous, exists := targetIndexes[target.Name]; exists {
			validationErrors = append(validationErrors, fmt.Errorf("target %d: duplicate target name %q (already used by target %d)", index, target.Name, previous))
		} else {
			targetIndexes[target.Name] = index
		}

		if len(target.Src) == 0 {
			validationErrors = append(validationErrors, fmt.Errorf("target %q: src must not be empty", target.Name))
		}
		if !knownTypes[target.Type] {
			validationErrors = append(validationErrors, fmt.Errorf("target %q: unknown type %q", target.Name, target.Type))
		}
		if _, exists := cfg.Toolchains[target.Lang]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("target %q: unknown lang %q", target.Name, target.Lang))
		} else if _, exists := cfg.Toolchains[target.Lang][target.Toolchain]; !exists {
			validationErrors = append(validationErrors, fmt.Errorf("target %q: unknown toolchain %q for lang %q", target.Name, target.Toolchain, target.Lang))
		}
		if target.Type != "static" {
			if _, exists := cfg.Linkers[target.Linker]; !exists {
				validationErrors = append(validationErrors, fmt.Errorf("target %q: unknown linker %q", target.Name, target.Linker))
			}
		}
		for _, dependency := range target.DependsOn {
			found := false
			for _, candidate := range cfg.Targets {
				if candidate.Name == dependency {
					found = true
					break
				}
			}
			if !found {
				validationErrors = append(validationErrors, fmt.Errorf("target %q: depends_on references unknown target %q", target.Name, dependency))
			}
		}
	}

	const (
		unvisited = iota
		visiting
		visited
	)
	states := make([]int, len(cfg.Targets))
	var visit func(int, []int)
	visit = func(index int, path []int) {
		states[index] = visiting
		path = append(path, index)
		for _, dependency := range cfg.Targets[index].DependsOn {
			dependencyIndex, exists := targetIndexes[dependency]
			if !exists {
				continue
			}
			switch states[dependencyIndex] {
			case unvisited:
				visit(dependencyIndex, path)
			case visiting:
				cycleStart := 0
				for cycleStart < len(path) && path[cycleStart] != dependencyIndex {
					cycleStart++
				}
				cycle := make([]string, 0, len(path)-cycleStart+1)
				for _, cycleIndex := range path[cycleStart:] {
					cycle = append(cycle, cfg.Targets[cycleIndex].Name)
				}
				cycle = append(cycle, cfg.Targets[dependencyIndex].Name)
				validationErrors = append(validationErrors, fmt.Errorf("dependency cycle: %s", strings.Join(cycle, " -> ")))
			}
		}
		states[index] = visited
	}
	for index := range cfg.Targets {
		if states[index] == unvisited {
			visit(index, nil)
		}
	}

	return validationErrors
}

func topoSort(targets []parser.Target) ([]parser.Target, error) {
	// Kahn's algorithm: https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
	index := make(map[string]parser.Target, len(targets))
	for _, t := range targets {
		index[t.Name] = t
	}

	inDegree := make(map[string]int, len(targets))
	for _, t := range targets {
		for _, dep := range t.DependsOn {
			inDegree[t.Name]++
			if _, ok := index[dep]; !ok {
				return nil, fmt.Errorf("target %s: depends_on %q does not exist", t.Name, dep)
			}
		}
	}

	var queue []parser.Target
	for _, t := range targets {
		if inDegree[t.Name] == 0 {
			queue = append(queue, t)
		}
	}

	var order []parser.Target
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		order = append(order, t)

		for _, dep := range t.DependsOn {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, index[dep])
			}
		}
	}

	if len(order) != len(targets) {
		return nil, fmt.Errorf("cycle detected in target dependencies")
	}

	return order, nil
}

func buildCompileRule(name string, tc parser.Toolchain) Rule {
	return Rule{
		Name:    name,
		Command: fmt.Sprintf("%s $flags -MMD -MF $out.d -c $in -o $out", tc.Run),
		Depfile: "$out.d",
		Deps:    "gcc",
	}
}

func linkRuleFor(t parser.Target, cfg *parser.Config) Rule {
	if t.Type == "static" {
		return Rule{
			Name:    "link_ar",
			Command: "ar rcs $out $flags $in $libs",
		}
	}

	linker, ok := cfg.Linkers[t.Linker]
	if !ok {
		panic(fmt.Sprintf("target %s: unknown linker %q", t.Name, t.Linker))
	}

	return Rule{
		Name:    fmt.Sprintf("link_%s", t.Linker),
		Command: fmt.Sprintf("%s $flags $in -o $out $libs", linker.Run),
	}
}

func outputPath(buildDir string, t parser.Target) string {
	if t.Type != "binary" && t.Type != "shared" && t.Type != "static" {
		panic(fmt.Sprintf("unknown target type %q", t.Type))
	}
	return fmt.Sprintf("%s/%s", buildDir, t.Outfile)
}

func objectPath(buildDir, targetName, src string) string {
	return fmt.Sprintf("%s/obj/%s/%s.o", buildDir, targetName, strings.TrimSuffix(src, ".c"))
}
