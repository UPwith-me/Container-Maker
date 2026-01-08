package workspace

import (
	"fmt"
	"sort"
	"strings"
)

// Graph represents a dependency graph for services
type Graph struct {
	nodes    map[string]*Node
	workspace *Workspace
}

// Node represents a service in the dependency graph
type Node struct {
	Name       string
	Service    *Service
	DependsOn  []string // Services this depends on
	Dependents []string // Services that depend on this
	Visited    bool
	InStack    bool
	Order      int // Topological order
}

// NewGraph creates a new dependency graph from a workspace
func NewGraph(ws *Workspace) (*Graph, error) {
	g := &Graph{
		nodes:     make(map[string]*Node),
		workspace: ws,
	}

	// Create nodes for each service
	for name, svc := range ws.Services {
		g.nodes[name] = &Node{
			Name:       name,
			Service:    svc,
			DependsOn:  svc.DependsOn,
			Dependents: []string{},
		}
	}

	// Build reverse dependencies (dependents)
	for name, node := range g.nodes {
		for _, dep := range node.DependsOn {
			if depNode, exists := g.nodes[dep]; exists {
				depNode.Dependents = append(depNode.Dependents, name)
			} else {
				return nil, fmt.Errorf("service %s depends on unknown service %s", name, dep)
			}
		}
	}

	return g, nil
}

// StartOrder returns services in the order they should be started
// Dependencies are started before their dependents
func (g *Graph) StartOrder() ([]string, error) {
	order, err := g.topologicalSort()
	if err != nil {
		return nil, err
	}
	return order, nil
}

// StopOrder returns services in the order they should be stopped
// Dependents are stopped before their dependencies (reverse of start order)
func (g *Graph) StopOrder() ([]string, error) {
	order, err := g.topologicalSort()
	if err != nil {
		return nil, err
	}
	// Reverse the order
	reversed := make([]string, len(order))
	for i, name := range order {
		reversed[len(order)-1-i] = name
	}
	return reversed, nil
}

// topologicalSort performs Kahn's algorithm for topological sorting
func (g *Graph) topologicalSort() ([]string, error) {
	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for name := range g.nodes {
		inDegree[name] = 0
	}
	for _, node := range g.nodes {
		for _, dep := range node.DependsOn {
			inDegree[node.Name]++
			_ = dep // dep is a dependency
		}
	}

	// Find all nodes with no dependencies (in-degree 0)
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // Deterministic order

	var result []string
	for len(queue) > 0 {
		// Pop from queue
		name := queue[0]
		queue = queue[1:]
		result = append(result, name)

		// Reduce in-degree of dependents
		node := g.nodes[name]
		for _, dependent := range node.Dependents {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
				sort.Strings(queue) // Keep sorted for determinism
			}
		}
	}

	// Check for cycles
	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

// GetDependencies returns all dependencies (direct and transitive) for a service
func (g *Graph) GetDependencies(serviceName string) ([]string, error) {
	node, exists := g.nodes[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	visited := make(map[string]bool)
	var deps []string

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		n := g.nodes[name]
		for _, dep := range n.DependsOn {
			visit(dep)
			deps = append(deps, dep)
		}
	}

	for _, dep := range node.DependsOn {
		visit(dep)
	}

	// Remove duplicates and sort
	unique := make(map[string]bool)
	var result []string
	for _, d := range deps {
		if !unique[d] {
			unique[d] = true
			result = append(result, d)
		}
	}
	sort.Strings(result)

	return result, nil
}

// GetDependents returns all services that depend on this service (direct and transitive)
func (g *Graph) GetDependents(serviceName string) ([]string, error) {
	node, exists := g.nodes[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	visited := make(map[string]bool)
	var dependents []string

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		n := g.nodes[name]
		for _, dep := range n.Dependents {
			dependents = append(dependents, dep)
			visit(dep)
		}
	}

	for _, dep := range node.Dependents {
		dependents = append(dependents, dep)
		visit(dep)
	}

	// Remove duplicates and sort
	unique := make(map[string]bool)
	var result []string
	for _, d := range dependents {
		if !unique[d] {
			unique[d] = true
			result = append(result, d)
		}
	}
	sort.Strings(result)

	return result, nil
}

// GetStartOrderForServices returns start order for specific services plus their dependencies
func (g *Graph) GetStartOrderForServices(services []string) ([]string, error) {
	needed := make(map[string]bool)

	// Add requested services and all their dependencies
	for _, svc := range services {
		needed[svc] = true
		deps, err := g.GetDependencies(svc)
		if err != nil {
			return nil, err
		}
		for _, dep := range deps {
			needed[dep] = true
		}
	}

	// Get full start order
	fullOrder, err := g.StartOrder()
	if err != nil {
		return nil, err
	}

	// Filter to only needed services
	var result []string
	for _, name := range fullOrder {
		if needed[name] {
			result = append(result, name)
		}
	}

	return result, nil
}

// GetStopOrderForServices returns stop order for specific services plus their dependents
func (g *Graph) GetStopOrderForServices(services []string) ([]string, error) {
	needed := make(map[string]bool)

	// Add requested services and all their dependents
	for _, svc := range services {
		needed[svc] = true
		deps, err := g.GetDependents(svc)
		if err != nil {
			return nil, err
		}
		for _, dep := range deps {
			needed[dep] = true
		}
	}

	// Get full stop order
	fullOrder, err := g.StopOrder()
	if err != nil {
		return nil, err
	}

	// Filter to only needed services
	var result []string
	for _, name := range fullOrder {
		if needed[name] {
			result = append(result, name)
		}
	}

	return result, nil
}

// Visualize returns a text representation of the dependency graph
func (g *Graph) Visualize() string {
	var sb strings.Builder
	sb.WriteString("Dependency Graph:\n")

	// Get services in start order
	order, err := g.StartOrder()
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error: %v\n", err))
		return sb.String()
	}

	for _, name := range order {
		node := g.nodes[name]
		sb.WriteString(fmt.Sprintf("  %s", name))
		if len(node.DependsOn) > 0 {
			sb.WriteString(fmt.Sprintf(" -> [%s]", strings.Join(node.DependsOn, ", ")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// Additional helper


