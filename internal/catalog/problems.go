package catalog

import tutiv1 "tuti-server/internal/genproto/tutiv1"

// Problems is the static practice-problem bank, keyed by id. Ported
// verbatim from MockTutiServerClient's problem builders.
var Problems = map[string]*tutiv1.Problem{
	"detected_linear_1": detectedLinearSystem(),
	"prob_linear_2":     linearSubstitutionPractice(),
	"prob_quad_1":       quadraticProblem(),
	"prob_pyth_1":       pythagoreanProblem(),
}

// System:  y = 2x + 1
//
//	y = -x + 4
//
// Solved by substitution -> intersection at (1, 3).
func detectedLinearSystem() *tutiv1.Problem {
	return &tutiv1.Problem{
		Id:         "detected_linear_1",
		Title:      "Detected: System of linear equations",
		Topic:      "Algebra",
		Difficulty: tutiv1.DifficultyLevel_DIFFICULTY_LEVEL_EASY,
		Statement: []*tutiv1.ContentBlock{
			text("The following system was found on your page. Solve by substitution:"),
			mathBlock(`y = 2x + 1`),
			mathBlock(`y = -x + 4`),
			graph2D("Graph — find the intersection",
				axis("x", -1, 5), axis("y", -2, 8),
				lineSeries("y = 2x + 1", "#6200EA", linePoints(-1, 3, 0.5, func(x float64) float64 { return 2*x + 1 }), false),
				lineSeries("y = −x + 4", "#E65100", linePoints(-1, 5, 0.5, func(x float64) float64 { return -x + 4 }), false),
			),
		},
		Hints: []string{
			"Both equations are already solved for y — set the right-hand sides equal.",
			"Once you find x, plug it back into either original equation to get y.",
		},
		Solution: []*tutiv1.ContentBlock{
			step(1, "Set the expressions equal",
				text("Both equations equal y, so substitute one into the other:"),
				mathBlock(`2x + 1 = -x + 4`),
			),
			step(2, "Solve for x",
				mathBlock(`2x + x = 4 - 1`),
				mathBlock(`3x = 3`),
				mathBlock(`x = 1`),
			),
			step(3, "Back-substitute to find y",
				text("Substitute x = 1 into the first equation:"),
				mathBlock(`y = 2(1) + 1 = 3`),
			),
			step(4, "Write the solution",
				mathBlock(`(x,\, y) = (1,\, 3)`),
				text("The two lines intersect at the point (1, 3) — visible on the graph above."),
			),
		},
	}
}

// System:  y = x + 2
//
//	y = 3x - 4
//
// Substitution -> intersection at (3, 5).
func linearSubstitutionPractice() *tutiv1.Problem {
	return &tutiv1.Problem{
		Id:         "prob_linear_2",
		Title:      "Solve by substitution",
		Topic:      "Algebra",
		Difficulty: tutiv1.DifficultyLevel_DIFFICULTY_LEVEL_EASY,
		Statement: []*tutiv1.ContentBlock{
			text("Solve the system using substitution:"),
			mathBlock(`y = x + 2`),
			mathBlock(`y = 3x - 4`),
		},
		Hints: []string{
			"Both equations are already isolated for y.",
			"After finding x, substitute back into y = x + 2.",
		},
		Solution: []*tutiv1.ContentBlock{
			step(1, "Set right-hand sides equal", mathBlock(`x + 2 = 3x - 4`)),
			step(2, "Solve for x",
				mathBlock(`2 + 4 = 3x - x`),
				mathBlock(`6 = 2x`),
				mathBlock(`x = 3`),
			),
			step(3, "Find y", mathBlock(`y = 3 + 2 = 5`)),
			step(4, "Solution", mathBlock(`(x,\, y) = (3,\, 5)`)),
		},
	}
}

// x^2 - 4x + 3 = 0
func quadraticProblem() *tutiv1.Problem {
	return &tutiv1.Problem{
		Id:         "prob_quad_1",
		Title:      "Solve the quadratic",
		Topic:      "Algebra",
		Difficulty: tutiv1.DifficultyLevel_DIFFICULTY_LEVEL_MEDIUM,
		Statement: []*tutiv1.ContentBlock{
			text("Solve for x:"),
			mathBlock(`x^2 - 4x + 3 = 0`),
			graph2D("Graph of f(x) = x² − 4x + 3",
				axis("x", -0.5, 4.5), axis("f(x)", -1.5, 5.5),
				lineSeries(`f(x) = x² − 4x + 3`, "#6200EA", parabolaPoints(-0.5, 4.5, 0.25, func(x float64) float64 { return x*x - 4*x + 3 }), false),
				lineSeries("x-axis", "#888888", []*tutiv1.Point2D{{X: -0.5, Y: 0}, {X: 4.5, Y: 0}}, true),
			),
		},
		Hints: []string{
			"Can you spot the x-intercepts on the graph?",
			"Try to factor: find two numbers that multiply to 3 and add to −4.",
		},
		Solution: []*tutiv1.ContentBlock{
			step(1, "Factor the expression",
				text("Look for two numbers that multiply to c = 3 and add to b = −4."),
				mathBlock(`x^2 - 4x + 3 = (x - 1)(x - 3)`),
			),
			step(2, "Apply the zero-product property",
				text("If a product equals zero, at least one factor is zero."),
				mathBlock(`x - 1 = 0 \quad \text{or} \quad x - 3 = 0`),
			),
			step(3, "Solve each equation", mathBlock(`x_1 = 1, \quad x_2 = 3`)),
		},
	}
}

// Right triangle, legs a = 3 and b = 4, find hypotenuse c.
func pythagoreanProblem() *tutiv1.Problem {
	return &tutiv1.Problem{
		Id:         "prob_pyth_1",
		Title:      "Find the hypotenuse",
		Topic:      "Geometry",
		Difficulty: tutiv1.DifficultyLevel_DIFFICULTY_LEVEL_EASY,
		Statement: []*tutiv1.ContentBlock{
			text("A right triangle has legs a = 3 and b = 4. Find c."),
			geometry("Right triangle", viewBox(-0.8, -0.7, 5.2, 4.4),
				segment(0, 0, 4, 0, "#000000"),
				segment(0, 0, 0, 3, "#000000"),
				segment(4, 0, 0, 3, "#6200EA"),
				geoAngle(0, 0, 0.4, 0, 90, "#0055FF"),
				geoLabel(2.0, -0.35, "a = 4", "#000000"),
				geoLabel(-0.45, 1.5, "b = 3", "#000000"),
				geoLabel(2.4, 1.9, "c = ?", "#6200EA"),
			),
		},
		Hints: []string{
			"The Pythagorean theorem: a² + b² = c²",
			`\sqrt{25} = 5`,
		},
		Solution: []*tutiv1.ContentBlock{
			step(1, "Write the Pythagorean theorem", mathBlock(`a^2 + b^2 = c^2`)),
			step(2, "Substitute the values",
				mathBlock(`3^2 + 4^2 = c^2`),
				mathBlock(`9 + 16 = c^2`),
				mathBlock(`c^2 = 25`),
			),
			step(3, "Solve for c", mathBlock(`c = \sqrt{25} = 5`)),
		},
	}
}
