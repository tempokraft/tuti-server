package analysis

// Hand-written JSON Schemas for the two forced tool calls. Kept flat and
// explicit rather than reflected off the draft structs — this is a small,
// stable shape and a hand-written schema is easier to reason about than
// fighting a reflection library's handling of enums/oneofs for such a
// small surface.
//
// Field descriptions live in prompts.go, not here — they're prompt text
// that steers the model, and prompts.go is the one place that owns prompt
// wording. This file only owns the schema *shape*.
//
// Every object below sets "additionalProperties": false. Anthropic enforces
// this exactly when the tool is marked Strict (see provider_anthropic.go).
// OpenAI's own strict mode would additionally require every property to be
// listed in "required" (optional fields expressed as nullable unions
// instead), which our optional text/math/hint fields don't fit without
// restructuring the schema — so provider_openai.go treats this key as
// best-effort guidance rather than a server-enforced guarantee there.
// Either way, draft.go's Validate() is the backstop that catches what
// schema validation alone can't (wrong enum value, empty required text).

func blockSchema() map[string]any {
	return map[string]any{
		"type":        "object",
		"description": descBlock,
		"properties": map[string]any{
			"kind":       map[string]any{"type": "string", "enum": []string{"text", "math", "hint"}},
			"text":       map[string]any{"type": "string", "description": descBlockText},
			"expression": map[string]any{"type": "string", "description": descBlockMath},
			"hint":       map[string]any{"type": "string", "description": descBlockHint},
		},
		"required":             []string{"kind"},
		"additionalProperties": false,
	}
}

func stepSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"stepNumber": map[string]any{"type": "integer"},
			"label":      map[string]any{"type": "string", "description": descStepLabel},
			"content":    map[string]any{"type": "array", "items": blockSchema()},
		},
		"required":             []string{"stepNumber", "label", "content"},
		"additionalProperties": false,
	}
}

func problemSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title":      map[string]any{"type": "string"},
			"topic":      map[string]any{"type": "string", "description": descProblemTopic},
			"difficulty": map[string]any{"type": "string", "enum": []string{"easy", "medium", "hard"}},
			"statement":  map[string]any{"type": "array", "items": blockSchema(), "description": descStatement},
			"hints":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"solution":   map[string]any{"type": "array", "items": stepSchema(), "description": descSolution},
		},
		"required":             []string{"title", "topic", "difficulty", "statement", "hints", "solution"},
		"additionalProperties": false,
	}
}

func mistakeSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description":   map[string]any{"type": "string", "description": descMistake},
			"stepReference": map[string]any{"type": "string", "description": descStepReference},
		},
		"required":             []string{"description"},
		"additionalProperties": false,
	}
}

// extractInputProperties/extractInputRequired and their evaluate
// counterparts are split into (properties, required) rather than a single
// wrapper object because the provider interface takes those two
// separately (each backend supplies its own "type": "object" wrapper).

func extractInputProperties() map[string]any {
	return map[string]any{
		"blank":   map[string]any{"type": "boolean", "description": descBlank},
		"problem": problemSchema(),
	}
}

func extractInputRequired() []string { return []string{"blank"} }

func evaluateInputProperties() map[string]any {
	return map[string]any{
		"problem": problemSchema(),
		"mistakes": map[string]any{
			"type":        "array",
			"items":       mistakeSchema(),
			"description": descMistakesList,
		},
	}
}

func evaluateInputRequired() []string { return []string{"problem", "mistakes"} }
