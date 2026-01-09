package detect

import (
	"sort"
	"strings"
)

// TemplateScorer scores and ranks template recommendations
type TemplateScorer struct {
	templates map[string]TemplateDefinition
}

// TemplateDefinition defines a template with its scoring criteria
type TemplateDefinition struct {
	Name        string
	Languages   []string
	Frameworks  []string
	Keywords    []string
	RequiresGPU bool
	Weight      float64
}

// ScoredTemplate represents a scored template recommendation
type ScoredTemplate struct {
	Name       string   `json:"name"`
	Score      float64  `json:"score"`
	Confidence string   `json:"confidence"` // "high", "medium", "low"
	Reasons    []string `json:"reasons"`
	MatchedBy  []string `json:"matchedBy"` // what triggered this match
}

// NewTemplateScorer creates a new template scorer
func NewTemplateScorer() *TemplateScorer {
	ts := &TemplateScorer{
		templates: make(map[string]TemplateDefinition),
	}
	ts.registerTemplates()
	return ts
}

// registerTemplates sets up the built-in templates
func (ts *TemplateScorer) registerTemplates() {
	// GPU/ML Templates (high weight)
	ts.templates["pytorch"] = TemplateDefinition{
		Name:        "pytorch",
		Languages:   []string{"Python"},
		Frameworks:  []string{"PyTorch"},
		Keywords:    []string{"torch", "pytorch", "cuda", "gpu", "neural", "deep learning"},
		RequiresGPU: true,
		Weight:      3.0,
	}

	ts.templates["tensorflow"] = TemplateDefinition{
		Name:        "tensorflow",
		Languages:   []string{"Python"},
		Frameworks:  []string{"TensorFlow"},
		Keywords:    []string{"tensorflow", "keras", "tf", "tpu"},
		RequiresGPU: true,
		Weight:      3.0,
	}

	ts.templates["jax-flax"] = TemplateDefinition{
		Name:        "jax-flax",
		Languages:   []string{"Python"},
		Frameworks:  []string{"JAX"},
		Keywords:    []string{"jax", "flax", "optax"},
		RequiresGPU: true,
		Weight:      3.0,
	}

	// Web Framework Templates
	ts.templates["nextjs"] = TemplateDefinition{
		Name:       "nextjs",
		Languages:  []string{"JavaScript", "TypeScript"},
		Frameworks: []string{"Next.js"},
		Keywords:   []string{"next", "nextjs", "vercel"},
		Weight:     2.5,
	}

	ts.templates["react"] = TemplateDefinition{
		Name:       "react",
		Languages:  []string{"JavaScript", "TypeScript"},
		Frameworks: []string{"React"},
		Keywords:   []string{"react", "jsx", "webpack"},
		Weight:     2.0,
	}

	ts.templates["vue"] = TemplateDefinition{
		Name:       "vue",
		Languages:  []string{"JavaScript", "TypeScript"},
		Frameworks: []string{"Vue", "Nuxt"},
		Keywords:   []string{"vue", "vuex", "pinia", "nuxt"},
		Weight:     2.0,
	}

	ts.templates["angular"] = TemplateDefinition{
		Name:       "angular",
		Languages:  []string{"TypeScript"},
		Frameworks: []string{"Angular"},
		Keywords:   []string{"angular", "ng"},
		Weight:     2.0,
	}

	// Python Web Frameworks
	ts.templates["python-django"] = TemplateDefinition{
		Name:       "python-django",
		Languages:  []string{"Python"},
		Frameworks: []string{"Django"},
		Keywords:   []string{"django"},
		Weight:     2.0,
	}

	ts.templates["python-fastapi"] = TemplateDefinition{
		Name:       "python-fastapi",
		Languages:  []string{"Python"},
		Frameworks: []string{"FastAPI"},
		Keywords:   []string{"fastapi", "starlette", "uvicorn"},
		Weight:     2.0,
	}

	ts.templates["python-flask"] = TemplateDefinition{
		Name:       "python-flask",
		Languages:  []string{"Python"},
		Frameworks: []string{"Flask"},
		Keywords:   []string{"flask"},
		Weight:     1.8,
	}

	// Node.js Frameworks
	ts.templates["nestjs"] = TemplateDefinition{
		Name:       "nestjs",
		Languages:  []string{"TypeScript"},
		Frameworks: []string{"NestJS"},
		Keywords:   []string{"nestjs", "@nestjs"},
		Weight:     2.0,
	}

	ts.templates["node-express"] = TemplateDefinition{
		Name:       "node-express",
		Languages:  []string{"JavaScript", "TypeScript"},
		Frameworks: []string{"Express"},
		Keywords:   []string{"express"},
		Weight:     1.8,
	}

	// Go Frameworks
	ts.templates["go-gin"] = TemplateDefinition{
		Name:       "go-gin",
		Languages:  []string{"Go"},
		Frameworks: []string{"Gin"},
		Keywords:   []string{"gin-gonic"},
		Weight:     2.0,
	}

	ts.templates["go-basic"] = TemplateDefinition{
		Name:      "go-basic",
		Languages: []string{"Go"},
		Keywords:  []string{"go.mod"},
		Weight:    1.5,
	}

	// Rust
	ts.templates["rust-basic"] = TemplateDefinition{
		Name:      "rust-basic",
		Languages: []string{"Rust"},
		Keywords:  []string{"cargo"},
		Weight:    1.5,
	}

	// Java
	ts.templates["java-spring"] = TemplateDefinition{
		Name:       "java-spring",
		Languages:  []string{"Java"},
		Frameworks: []string{"Spring"},
		Keywords:   []string{"spring", "springboot"},
		Weight:     2.0,
	}

	ts.templates["java-maven"] = TemplateDefinition{
		Name:      "java-maven",
		Languages: []string{"Java"},
		Keywords:  []string{"pom.xml", "maven"},
		Weight:    1.5,
	}

	ts.templates["java-gradle"] = TemplateDefinition{
		Name:      "java-gradle",
		Languages: []string{"Java", "Kotlin"},
		Keywords:  []string{"build.gradle", "gradle"},
		Weight:    1.5,
	}

	// .NET
	ts.templates["dotnet"] = TemplateDefinition{
		Name:      "dotnet",
		Languages: []string{".NET/C#", ".NET/F#", ".NET"},
		Keywords:  []string{"csproj", "dotnet", "aspnet"},
		Weight:    1.5,
	}

	// C++
	ts.templates["cpp-cmake"] = TemplateDefinition{
		Name:      "cpp-cmake",
		Languages: []string{"C++", "C"},
		Keywords:  []string{"cmake", "CMakeLists"},
		Weight:    1.5,
	}

	// Default templates
	ts.templates["python-basic"] = TemplateDefinition{
		Name:      "python-basic",
		Languages: []string{"Python"},
		Keywords:  []string{"requirements.txt", "pyproject.toml", "setup.py"},
		Weight:    1.0,
	}

	ts.templates["node-basic"] = TemplateDefinition{
		Name:      "node-basic",
		Languages: []string{"JavaScript", "TypeScript"},
		Keywords:  []string{"package.json"},
		Weight:    1.0,
	}
}

// ScoreTemplates scores all templates against project info
func (ts *TemplateScorer) ScoreTemplates(info *ProjectInfo) []ScoredTemplate {
	scores := make(map[string]*ScoredTemplate)

	for name, tmpl := range ts.templates {
		score, reasons, matchedBy := ts.scoreTemplate(tmpl, info)
		if score > 0 {
			scores[name] = &ScoredTemplate{
				Name:      name,
				Score:     score,
				Reasons:   reasons,
				MatchedBy: matchedBy,
			}
		}
	}

	// Convert to slice and sort
	var result []ScoredTemplate
	for _, st := range scores {
		// Determine confidence
		if st.Score >= 2.0 {
			st.Confidence = "high"
		} else if st.Score >= 1.0 {
			st.Confidence = "medium"
		} else {
			st.Confidence = "low"
		}
		result = append(result, *st)
	}

	// Sort by score (descending)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	// Return top 5
	if len(result) > 5 {
		result = result[:5]
	}

	return result
}

// scoreTemplate calculates score for a single template
func (ts *TemplateScorer) scoreTemplate(tmpl TemplateDefinition, info *ProjectInfo) (float64, []string, []string) {
	var score float64
	var reasons []string
	var matchedBy []string

	// Language matching
	for _, lang := range tmpl.Languages {
		if containsLanguageCI(info.Languages, lang) {
			score += 0.5 * tmpl.Weight
			reasons = append(reasons, "Language match: "+lang)
			matchedBy = append(matchedBy, "language:"+lang)
		}
	}

	// Framework matching (high value)
	for _, fw := range tmpl.Frameworks {
		if containsCI(info.Frameworks, fw) {
			score += 1.0 * tmpl.Weight
			reasons = append(reasons, "Framework match: "+fw)
			matchedBy = append(matchedBy, "framework:"+fw)
		}
	}

	// Keyword matching in dependencies
	for _, kw := range tmpl.Keywords {
		for _, dep := range info.Dependencies {
			if strings.Contains(strings.ToLower(dep), strings.ToLower(kw)) {
				score += 0.3 * tmpl.Weight
				reasons = append(reasons, "Dependency match: "+kw)
				matchedBy = append(matchedBy, "dependency:"+kw)
				break
			}
		}
	}

	// GPU requirement matching
	if tmpl.RequiresGPU && info.NeedsGPU {
		score += 1.5 * tmpl.Weight
		reasons = append(reasons, "GPU requirement match")
		matchedBy = append(matchedBy, "gpu")
	} else if tmpl.RequiresGPU && !info.NeedsGPU {
		// Penalize GPU templates for non-GPU projects
		score *= 0.3
	}

	// Version match bonus
	if info.Versions != nil {
		for lang := range info.Versions {
			for _, tl := range tmpl.Languages {
				if strings.EqualFold(lang, tl) {
					score += 0.2
					reasons = append(reasons, "Version specified: "+lang)
				}
			}
		}
	}

	return score, reasons, matchedBy
}

// GetRecommendation returns the top recommendation with explanation
func (ts *TemplateScorer) GetRecommendation(info *ProjectInfo) (*ScoredTemplate, string) {
	scores := ts.ScoreTemplates(info)

	if len(scores) == 0 {
		return nil, "No matching template found"
	}

	top := &scores[0]

	// Build explanation
	var explanation strings.Builder
	explanation.WriteString("Recommended: ")
	explanation.WriteString(top.Name)
	explanation.WriteString(" (")
	explanation.WriteString(top.Confidence)
	explanation.WriteString(" confidence)\n")
	explanation.WriteString("Reasons:\n")
	for _, r := range top.Reasons {
		explanation.WriteString("  • ")
		explanation.WriteString(r)
		explanation.WriteString("\n")
	}

	if len(scores) > 1 {
		explanation.WriteString("\nAlternatives:\n")
		for i := 1; i < len(scores) && i <= 3; i++ {
			explanation.WriteString("  ")
			explanation.WriteString(scores[i].Name)
			explanation.WriteString(" (")
			explanation.WriteString(scores[i].Confidence)
			explanation.WriteString(")\n")
		}
	}

	return top, explanation.String()
}

// Helper functions
func containsCI(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func containsLanguageCI(langs []LanguageInfo, name string) bool {
	for _, l := range langs {
		if strings.EqualFold(l.Name, name) {
			return true
		}
	}
	return false
}
