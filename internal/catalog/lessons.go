package catalog

import tutiv1 "tuti-server/internal/genproto/tutiv1"

// Lessons is the static lesson library, keyed by lesson id, each entry a
// constructor taking a BCP-47-ish language code ("en"/"es"). Ported
// verbatim from MockTutiServerClient's lesson builders.
var Lessons = map[string]func(lang string) *tutiv1.LessonContent{
	"linear_equations":       linearEquationsLesson,
	"quadratic_equations":    quadraticEquationsLesson,
	"pythagorean_theorem":    pythagoreanTheoremLesson,
	"intro_to_derivatives":   introToDerivativesLesson,
	"statistics_mean_median": statisticsLesson,
}

// GetLesson looks up a lesson by id and language, falling back to a stub
// "coming soon" lesson for unknown ids (matching the mock's behavior).
func GetLesson(id, lang string) *tutiv1.LessonContent {
	if fn, ok := Lessons[id]; ok {
		return fn(lang)
	}
	return fallbackLesson(id)
}

func fallbackLesson(id string) *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id:       id,
		Title:    id,
		Topic:    "Mathematics",
		IconName: "school_outlined",
		Language: "en",
		Context:  []*tutiv1.ContentBlock{text("Lesson content coming soon.")},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Example",
			Content: []*tutiv1.ContentBlock{text("No example available yet.")},
		},
	}
}

// ── Linear Equations ─────────────────────────────────────────────────────

func linearEquationsLesson(lang string) *tutiv1.LessonContent {
	if lang == "es" {
		return linearEquationsEs()
	}
	return linearEquationsEn()
}

func linearEquationsEn() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "linear_equations", Title: "Linear Equations", Topic: "Algebra",
		IconName: "linear_scale", Language: "en",
		Context: []*tutiv1.ContentBlock{
			text("Linear equations describe straight-line relationships between quantities. They appear constantly in daily life: splitting a bill, converting temperatures, calculating travel time."),
			text("Any time you know two quantities are equal and one is unknown, you are facing a linear equation."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "What is a Linear Equation?",
				Content: []*tutiv1.ContentBlock{
					text("A linear equation in one variable has the standard form:"),
					mathBlock(`ax + b = c`),
					text("where a ≠ 0 is the coefficient of the unknown x, and b and c are constants. Solving the equation means finding the value of x that makes the statement true."),
				},
			},
			{
				Heading: "Solving by Isolation",
				Content: []*tutiv1.ContentBlock{
					text("The goal is to get x alone on one side. Apply the same operation to both sides to keep the equation balanced."),
					step(1, "Subtract b from both sides", mathBlock(`ax + b - b = c - b`), mathBlock(`ax = c - b`)),
					step(2, "Divide both sides by a", mathBlock(`x = \frac{c - b}{a}`)),
				},
			},
			{
				Heading: "Solving Systems by Substitution",
				Content: []*tutiv1.ContentBlock{
					text("When two equations share the same two unknowns, express one variable in terms of the other, then substitute."),
					text("Example system:"),
					mathBlock(`y = 2x + 1`),
					mathBlock(`y = -x + 4`),
					graph2D("Both lines — where do they meet?", axis("x", -0.5, 4.5), axis("y", -1, 7),
						lineSeries("y = 2x + 1", "#6200EA", linePoints(-0.5, 2.5, 0.25, func(x float64) float64 { return 2*x + 1 }), false),
						lineSeries("y = −x + 4", "#E65100", linePoints(-0.5, 4.5, 0.25, func(x float64) float64 { return -x + 4 }), false),
					),
					step(1, "Substitute — set the right-hand sides equal", mathBlock(`2x + 1 = -x + 4`)),
					step(2, "Collect like terms", mathBlock(`3x = 3 \quad\Rightarrow\quad x = 1`)),
					step(3, "Back-substitute to find y", mathBlock(`y = 2(1) + 1 = 3`), mathBlock(`\text{Solution: } (x,\, y) = (1,\, 3)`)),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Worked Example",
			Content: []*tutiv1.ContentBlock{
				text("Solve for x:"),
				mathBlock(`5x - 3 = 12`),
				step(1, "Add 3 to both sides", mathBlock(`5x = 15`)),
				step(2, "Divide both sides by 5", mathBlock(`x = \frac{15}{5} = 3`)),
				text("Check: 5(3) − 3 = 15 − 3 = 12 ✓"),
			},
		},
	}
}

func linearEquationsEs() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "linear_equations", Title: "Ecuaciones Lineales", Topic: "Álgebra",
		IconName: "linear_scale", Language: "es",
		Context: []*tutiv1.ContentBlock{
			text("Las ecuaciones lineales describen relaciones de línea recta entre cantidades. Aparecen constantemente en la vida diaria: dividir una cuenta, convertir temperaturas, calcular tiempos de viaje."),
			text("Cada vez que sabes que dos cantidades son iguales y una es desconocida, estás frente a una ecuación lineal."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "¿Qué es una Ecuación Lineal?",
				Content: []*tutiv1.ContentBlock{
					text("Una ecuación lineal en una variable tiene la forma estándar:"),
					mathBlock(`ax + b = c`),
					text("donde a ≠ 0 es el coeficiente de la incógnita x, y b y c son constantes. Resolver la ecuación significa encontrar el valor de x que hace verdadera la igualdad."),
				},
			},
			{
				Heading: "Resolver por Aislamiento",
				Content: []*tutiv1.ContentBlock{
					text("El objetivo es dejar x solo en un lado. Aplica la misma operación a ambos lados para mantener el equilibrio."),
					step(1, "Restar b en ambos lados", mathBlock(`ax + b - b = c - b`), mathBlock(`ax = c - b`)),
					step(2, "Dividir ambos lados entre a", mathBlock(`x = \frac{c - b}{a}`)),
				},
			},
			{
				Heading: "Resolver Sistemas por Sustitución",
				Content: []*tutiv1.ContentBlock{
					text("Cuando dos ecuaciones comparten dos incógnitas, expresa una variable en términos de la otra y sustituye."),
					text("Sistema de ejemplo:"),
					mathBlock(`y = 2x + 1`),
					mathBlock(`y = -x + 4`),
					graph2D("Las dos rectas — ¿dónde se cortan?", axis("x", -0.5, 4.5), axis("y", -1, 7),
						lineSeries("y = 2x + 1", "#6200EA", linePoints(-0.5, 2.5, 0.25, func(x float64) float64 { return 2*x + 1 }), false),
						lineSeries("y = −x + 4", "#E65100", linePoints(-0.5, 4.5, 0.25, func(x float64) float64 { return -x + 4 }), false),
					),
					step(1, "Sustituir — igualar los lados derechos", mathBlock(`2x + 1 = -x + 4`)),
					step(2, "Reunir términos semejantes", mathBlock(`3x = 3 \quad\Rightarrow\quad x = 1`)),
					step(3, "Retro-sustituir para hallar y", mathBlock(`y = 2(1) + 1 = 3`), mathBlock(`\text{Solución: } (x,\, y) = (1,\, 3)`)),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Ejemplo Resuelto",
			Content: []*tutiv1.ContentBlock{
				text("Resuelve para x:"),
				mathBlock(`5x - 3 = 12`),
				step(1, "Suma 3 a ambos lados", mathBlock(`5x = 15`)),
				step(2, "Divide ambos lados entre 5", mathBlock(`x = \frac{15}{5} = 3`)),
				text("Comprobación: 5(3) − 3 = 15 − 3 = 12 ✓"),
			},
		},
	}
}

// ── Quadratic Equations ───────────────────────────────────────────────────

func quadraticEquationsLesson(lang string) *tutiv1.LessonContent {
	if lang == "es" {
		return quadraticEquationsEs()
	}
	return quadraticEquationsEn()
}

func quadraticEquationsEn() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "quadratic_equations", Title: "Quadratic Equations", Topic: "Algebra",
		IconName: "functions", Language: "en",
		Context: []*tutiv1.ContentBlock{
			text("Quadratic equations model projectile motion, areas, and profit curves. Any parabola you see on a graph is a quadratic relationship."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "Standard Form",
				Content: []*tutiv1.ContentBlock{
					mathBlock(`ax^2 + bx + c = 0 \quad (a \neq 0)`),
					text("The graph is a parabola. When a > 0 it opens upward; when a < 0 it opens downward."),
				},
			},
			{
				Heading: "The Quadratic Formula",
				Content: []*tutiv1.ContentBlock{
					text("For any quadratic equation the solutions are:"),
					mathBlock(`x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}`),
					text("The expression b² − 4ac is the discriminant (Δ):"),
					mathBlock(`\Delta > 0 \Rightarrow \text{two real roots}`),
					mathBlock(`\Delta = 0 \Rightarrow \text{one real root}`),
					mathBlock(`\Delta < 0 \Rightarrow \text{no real roots}`),
					graph2D("f(x) = x² − 4x + 3  (roots at x = 1 and x = 3)", axis("x", -0.5, 4.5), axis("f(x)", -1.5, 5.5),
						lineSeries("f(x) = x² − 4x + 3", "#6200EA", parabolaPoints(-0.5, 4.5, 0.25, func(x float64) float64 { return x*x - 4*x + 3 }), false),
					),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Worked Example",
			Content: []*tutiv1.ContentBlock{
				text("Solve:"),
				mathBlock(`x^2 - 4x + 3 = 0`),
				step(1, "Identify a = 1, b = −4, c = 3", mathBlock(`\Delta = (-4)^2 - 4(1)(3) = 16 - 12 = 4`)),
				step(2, "Apply the formula", mathBlock(`x = \frac{4 \pm \sqrt{4}}{2} = \frac{4 \pm 2}{2}`)),
				step(3, "Two solutions", mathBlock(`x_1 = 3, \quad x_2 = 1`)),
			},
		},
	}
}

func quadraticEquationsEs() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "quadratic_equations", Title: "Ecuaciones Cuadráticas", Topic: "Álgebra",
		IconName: "functions", Language: "es",
		Context: []*tutiv1.ContentBlock{
			text("Las ecuaciones cuadráticas modelan el movimiento de proyectiles, áreas y curvas de ganancia. Toda parábola que ves en una gráfica es una relación cuadrática."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "Forma Estándar",
				Content: []*tutiv1.ContentBlock{
					mathBlock(`ax^2 + bx + c = 0 \quad (a \neq 0)`),
					text("La gráfica es una parábola. Si a > 0 abre hacia arriba; si a < 0 abre hacia abajo."),
				},
			},
			{
				Heading: "La Fórmula Cuadrática",
				Content: []*tutiv1.ContentBlock{
					text("Para cualquier ecuación cuadrática las soluciones son:"),
					mathBlock(`x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}`),
					text("La expresión b² − 4ac es el discriminante (Δ):"),
					mathBlock(`\Delta > 0 \Rightarrow \text{dos raíces reales}`),
					mathBlock(`\Delta = 0 \Rightarrow \text{una raíz real}`),
					mathBlock(`\Delta < 0 \Rightarrow \text{sin raíces reales}`),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Ejemplo Resuelto",
			Content: []*tutiv1.ContentBlock{
				text("Resuelve:"),
				mathBlock(`x^2 - 4x + 3 = 0`),
				step(1, "Identifica a = 1, b = −4, c = 3", mathBlock(`\Delta = 16 - 12 = 4`)),
				step(2, "Aplica la fórmula", mathBlock(`x = \frac{4 \pm 2}{2}`)),
				step(3, "Dos soluciones", mathBlock(`x_1 = 3, \quad x_2 = 1`)),
			},
		},
	}
}

// ── Pythagorean Theorem ───────────────────────────────────────────────────

func pythagoreanTheoremLesson(lang string) *tutiv1.LessonContent {
	if lang == "es" {
		return pythagoreanTheoremEs()
	}
	return pythagoreanTheoremEn()
}

func pythagoreanTheoremEn() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "pythagorean_theorem", Title: "Pythagorean Theorem", Topic: "Geometry",
		IconName: "change_history", Language: "en",
		Context: []*tutiv1.ContentBlock{
			text("Builders, sailors, and engineers use the Pythagorean theorem daily. It is the mathematical reason a 3-4-5 rope trick guarantees a right angle on a construction site."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "The Theorem",
				Content: []*tutiv1.ContentBlock{
					text("In any right triangle with legs a, b and hypotenuse c:"),
					mathBlock(`a^2 + b^2 = c^2`),
					geometry("Right triangle", viewBox(-0.8, -0.7, 5.2, 4.4),
						segment(0, 0, 4, 0, "#000000"),
						segment(0, 0, 0, 3, "#000000"),
						segment(4, 0, 0, 3, "#6200EA"),
						geoAngle(0, 0, 0.4, 0, 90, "#0055FF"),
						geoLabel(2.0, -0.35, "a", "#000000"),
						geoLabel(-0.3, 1.5, "b", "#000000"),
						geoLabel(2.4, 1.9, "c", "#6200EA"),
					),
				},
			},
			{
				Heading: "Finding the Hypotenuse",
				Content: []*tutiv1.ContentBlock{
					text("Rearrange to isolate c:"),
					mathBlock(`c = \sqrt{a^2 + b^2}`),
					text("Finding a leg when the other two sides are known:"),
					mathBlock(`a = \sqrt{c^2 - b^2}`),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Worked Example",
			Content: []*tutiv1.ContentBlock{
				text("A right triangle has legs a = 5 and b = 12. Find c."),
				step(1, "Square the legs", mathBlock(`5^2 + 12^2 = 25 + 144 = 169`)),
				step(2, "Take the square root", mathBlock(`c = \sqrt{169} = 13`)),
			},
		},
	}
}

func pythagoreanTheoremEs() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "pythagorean_theorem", Title: "Teorema de Pitágoras", Topic: "Geometría",
		IconName: "change_history", Language: "es",
		Context: []*tutiv1.ContentBlock{
			text("Constructores, marineros e ingenieros usan el teorema de Pitágoras a diario. Es la razón matemática por la que el truco de la cuerda 3-4-5 garantiza un ángulo recto."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "El Teorema",
				Content: []*tutiv1.ContentBlock{
					text("En todo triángulo rectángulo con catetos a, b e hipotenusa c:"),
					mathBlock(`a^2 + b^2 = c^2`),
					geometry("Triángulo rectángulo", viewBox(-0.8, -0.7, 5.2, 4.4),
						segment(0, 0, 4, 0, "#000000"),
						segment(0, 0, 0, 3, "#000000"),
						segment(4, 0, 0, 3, "#6200EA"),
						geoAngle(0, 0, 0.4, 0, 90, "#0055FF"),
						geoLabel(2.0, -0.35, "a", "#000000"),
						geoLabel(-0.3, 1.5, "b", "#000000"),
						geoLabel(2.4, 1.9, "c", "#6200EA"),
					),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Ejemplo Resuelto",
			Content: []*tutiv1.ContentBlock{
				text("Un triángulo rectángulo tiene catetos a = 5 y b = 12. Halla c."),
				step(1, "Elevar al cuadrado los catetos", mathBlock(`5^2 + 12^2 = 25 + 144 = 169`)),
				step(2, "Sacar la raíz cuadrada", mathBlock(`c = \sqrt{169} = 13`)),
			},
		},
	}
}

// ── Introduction to Derivatives ───────────────────────────────────────────

func introToDerivativesLesson(lang string) *tutiv1.LessonContent {
	if lang == "es" {
		return introToDerivativesEs()
	}
	return introToDerivativesEn()
}

func introToDerivativesEn() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "intro_to_derivatives", Title: "Introduction to Derivatives", Topic: "Calculus",
		IconName: "show_chart", Language: "en",
		Context: []*tutiv1.ContentBlock{
			text("The derivative answers one question: how fast is this changing right now? Speedometers, weather models, and economic forecasts all depend on rates of change."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "Definition",
				Content: []*tutiv1.ContentBlock{
					text("The derivative of f at x is the slope of the tangent line:"),
					mathBlock(`f'(x) = \lim_{h \to 0} \frac{f(x+h) - f(x)}{h}`),
					graph2D("f(x) = x² and its tangent at x = 1  (slope = 2)", axis("x", -2, 3), axis("f(x)", -1, 6),
						lineSeries("f(x) = x²", "#6200EA", parabolaPoints(-2, 3, 0.2, func(x float64) float64 { return x * x }), false),
						lineSeries("Tangent at x = 1  (f'(1) = 2)", "#E65100", linePoints(-0.5, 2.5, 0.5, func(x float64) float64 { return 2*x - 1 }), true),
					),
				},
			},
			{
				Heading: "Power Rule",
				Content: []*tutiv1.ContentBlock{
					text("The most used differentiation rule: for f(x) = xⁿ,"),
					mathBlock(`f'(x) = n \cdot x^{n-1}`),
					text("Examples:"),
					mathBlock(`f(x) = x^3 \Rightarrow f'(x) = 3x^2`),
					mathBlock(`f(x) = x^5 \Rightarrow f'(x) = 5x^4`),
					mathBlock(`f(x) = x \Rightarrow f'(x) = 1`),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Worked Example",
			Content: []*tutiv1.ContentBlock{
				text("Differentiate f(x) = 4x³ − 2x + 7."),
				step(1, "Apply the power rule term by term",
					mathBlock(`\frac{d}{dx}[4x^3] = 12x^2`),
					mathBlock(`\frac{d}{dx}[-2x] = -2`),
					mathBlock(`\frac{d}{dx}[7] = 0`),
				),
				step(2, "Combine", mathBlock(`f'(x) = 12x^2 - 2`)),
			},
		},
	}
}

func introToDerivativesEs() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "intro_to_derivatives", Title: "Introducción a las Derivadas", Topic: "Cálculo",
		IconName: "show_chart", Language: "es",
		Context: []*tutiv1.ContentBlock{
			text("La derivada responde una pregunta: ¿qué tan rápido cambia esto ahora mismo? Los velocímetros, modelos meteorológicos y pronósticos económicos dependen de las tasas de cambio."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "Definición",
				Content: []*tutiv1.ContentBlock{
					text("La derivada de f en x es la pendiente de la recta tangente:"),
					mathBlock(`f'(x) = \lim_{h \to 0} \frac{f(x+h) - f(x)}{h}`),
				},
			},
			{
				Heading: "Regla de la Potencia",
				Content: []*tutiv1.ContentBlock{
					text("La regla de diferenciación más usada: para f(x) = xⁿ,"),
					mathBlock(`f'(x) = n \cdot x^{n-1}`),
					text("Ejemplos:"),
					mathBlock(`f(x) = x^3 \Rightarrow f'(x) = 3x^2`),
					mathBlock(`f(x) = x^5 \Rightarrow f'(x) = 5x^4`),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Ejemplo Resuelto",
			Content: []*tutiv1.ContentBlock{
				text("Deriva f(x) = 4x³ − 2x + 7."),
				step(1, "Aplica la regla de la potencia término a término", mathBlock(`f'(x) = 12x^2 - 2`)),
			},
		},
	}
}

// ── Statistics: Mean & Median ─────────────────────────────────────────────

func statisticsLesson(lang string) *tutiv1.LessonContent {
	if lang == "es" {
		return statisticsEs()
	}
	return statisticsEn()
}

func statisticsEn() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "statistics_mean_median", Title: "Mean & Median", Topic: "Statistics",
		IconName: "bar_chart", Language: "en",
		Context: []*tutiv1.ContentBlock{
			text("Averages summarise data. News headlines, sports statistics, and scientific studies all rely on mean and median to tell the story of a dataset."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "Mean (Average)",
				Content: []*tutiv1.ContentBlock{
					text("Add all values and divide by the count:"),
					mathBlock(`\bar{x} = \frac{x_1 + x_2 + \cdots + x_n}{n}`),
				},
			},
			{
				Heading: "Median",
				Content: []*tutiv1.ContentBlock{
					text("The middle value when data are sorted. If n is even, the median is the mean of the two middle values."),
				},
			},
			{
				Heading: "When to Use Each",
				Content: []*tutiv1.ContentBlock{
					text("The mean is sensitive to outliers. Use the median when the data contains extreme values (e.g. income data)."),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Worked Example",
			Content: []*tutiv1.ContentBlock{
				text("Dataset: 4, 7, 2, 9, 4, 6"),
				step(1, "Mean", mathBlock(`\bar{x} = \frac{4+7+2+9+4+6}{6} = \frac{32}{6} \approx 5.33`)),
				step(2, "Median — sort first: 2, 4, 4, 6, 7, 9", mathBlock(`\text{Median} = \frac{4 + 6}{2} = 5`)),
			},
		},
	}
}

func statisticsEs() *tutiv1.LessonContent {
	return &tutiv1.LessonContent{
		Id: "statistics_mean_median", Title: "Media y Mediana", Topic: "Estadística",
		IconName: "bar_chart", Language: "es",
		Context: []*tutiv1.ContentBlock{
			text("Los promedios resumen datos. Titulares de noticias, estadísticas deportivas y estudios científicos dependen de la media y la mediana para contar la historia de un conjunto de datos."),
		},
		Sections: []*tutiv1.LessonSection{
			{
				Heading: "Media (Promedio)",
				Content: []*tutiv1.ContentBlock{
					text("Suma todos los valores y divide entre la cantidad:"),
					mathBlock(`\bar{x} = \frac{x_1 + x_2 + \cdots + x_n}{n}`),
				},
			},
			{
				Heading: "Mediana",
				Content: []*tutiv1.ContentBlock{
					text("El valor central al ordenar los datos. Si n es par, la mediana es el promedio de los dos valores centrales."),
				},
			},
		},
		WorkedExample: &tutiv1.LessonSection{
			Heading: "Ejemplo Resuelto",
			Content: []*tutiv1.ContentBlock{
				text("Datos: 4, 7, 2, 9, 4, 6"),
				step(1, "Media", mathBlock(`\bar{x} = \frac{32}{6} \approx 5.33`)),
				step(2, "Mediana — ordenar: 2, 4, 4, 6, 7, 9", mathBlock(`\text{Mediana} = \frac{4 + 6}{2} = 5`)),
			},
		},
	}
}
