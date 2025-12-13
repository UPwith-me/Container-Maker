package template

import (
	"strings"
	"testing"
)

func TestGetTemplate(t *testing.T) {
	// Test getting a known template
	t.Run("KnownTemplate", func(t *testing.T) {
		tmpl, found := GetTemplate("python-basic")
		if !found {
			t.Fatal("Template 'python-basic' not found")
		}

		if tmpl.Name != "python-basic" {
			t.Errorf("Expected name 'python-basic', got '%s'", tmpl.Name)
		}

		if tmpl.Image == "" {
			t.Error("Expected image to be set")
		}
	})

	// Test unknown template
	t.Run("UnknownTemplate", func(t *testing.T) {
		_, found := GetTemplate("non-existent-template")
		if found {
			t.Error("Expected template to not be found")
		}
	})
}

func TestListTemplates(t *testing.T) {
	output := ListTemplates()

	if output == "" {
		t.Error("Expected non-empty template list")
	}

	// Should contain some known templates
	if !strings.Contains(output, "python") && !strings.Contains(output, "Python") {
		t.Error("Expected Python template in list")
	}
}

func TestGetAllTemplates(t *testing.T) {
	templates := GetAllTemplates()

	if len(templates) == 0 {
		t.Error("Expected at least one template")
	}

	// Check that all templates have required fields
	for name, tmpl := range templates {
		if tmpl.Name == "" {
			t.Errorf("Template '%s' has empty name", name)
		}
		if tmpl.Image == "" {
			t.Errorf("Template '%s' has empty image", name)
		}
		if tmpl.Category == "" {
			t.Errorf("Template '%s' has empty category", name)
		}
	}
}

func TestSearchTemplates(t *testing.T) {
	// Test search by query
	t.Run("SearchByQuery", func(t *testing.T) {
		results := SearchTemplates(SearchOptions{
			Query: "python",
		})

		if len(results) == 0 {
			t.Error("Expected at least one Python template")
		}

		for _, tmpl := range results {
			// Should match name, description, or category (case insensitive)
			name := strings.ToLower(tmpl.Name)
			desc := strings.ToLower(tmpl.Description)
			cat := strings.ToLower(tmpl.Category)
			if !strings.Contains(name, "python") &&
				!strings.Contains(desc, "python") &&
				!strings.Contains(cat, "python") {
				t.Errorf("Result '%s' doesn't match 'python' query", tmpl.Name)
			}
		}
	})

	// Test search by category
	t.Run("SearchByCategory", func(t *testing.T) {
		results := SearchTemplates(SearchOptions{
			Category: "Go",
		})

		if len(results) == 0 {
			t.Error("Expected at least one Go template")
		}

		for _, tmpl := range results {
			if tmpl.Category != "Go" {
				t.Errorf("Expected category 'Go', got '%s'", tmpl.Category)
			}
		}
	})

	// Test GPU filter
	t.Run("SearchGPUOnly", func(t *testing.T) {
		results := SearchTemplates(SearchOptions{
			GPUOnly: true,
		})

		for _, tmpl := range results {
			if !tmpl.RequiresGPU() {
				t.Errorf("Template '%s' doesn't require GPU", tmpl.Name)
			}
		}
	})
}

func TestGetCategories(t *testing.T) {
	categories := GetCategories()

	if len(categories) == 0 {
		t.Error("Expected at least one category")
	}

	// Check for common categories
	hasGo := false
	hasPython := false
	for _, cat := range categories {
		if cat == "Go" {
			hasGo = true
		}
		if cat == "Python" {
			hasPython = true
		}
	}

	if !hasGo {
		t.Error("Expected 'Go' category")
	}
	if !hasPython {
		t.Error("Expected 'Python' category")
	}
}

func TestRequiresGPU(t *testing.T) {
	// Test a template without GPU
	t.Run("NoGPU", func(t *testing.T) {
		tmpl := &Template{
			Name:     "test",
			Image:    "golang:1.21",
			Category: "Go",
		}
		if tmpl.RequiresGPU() {
			t.Error("Expected RequiresGPU() to return false")
		}
	})

	// Test a template with --gpus in RunArgs
	t.Run("GPUInRunArgs", func(t *testing.T) {
		tmpl := &Template{
			Name:    "pytorch-gpu",
			Image:   "nvidia/cuda",
			RunArgs: []string{"--gpus", "all"},
		}
		if !tmpl.RequiresGPU() {
			t.Error("Expected RequiresGPU() to return true for --gpus in RunArgs")
		}
	})

	// Test a template with Deep Learning category
	t.Run("DeepLearningCategory", func(t *testing.T) {
		tmpl := &Template{
			Name:     "ml-training",
			Image:    "tensorflow",
			Category: "Deep Learning",
		}
		if !tmpl.RequiresGPU() {
			t.Error("Expected RequiresGPU() to return true for Deep Learning category")
		}
	})
}
