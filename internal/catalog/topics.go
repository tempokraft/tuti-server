package catalog

import (
	"strings"

	tutiv1 "tuti-server/internal/genproto/tutiv1"
)

// Topics is the static topic-recommendation list shown for a blank-page
// capture. Ported verbatim from MockTutiServerClient's
// _topicRecommendations.
var Topics = []*tutiv1.TopicRecommendation{
	{Id: "quadratic", Title: "Quadratic Equations", Description: "Solve equations of the form ax² + bx + c = 0", LessonId: "quadratic_equations", IconName: "functions"},
	{Id: "pythagoras", Title: "Pythagorean Theorem", Description: "Relate the sides of a right triangle: a² + b² = c²", LessonId: "pythagorean_theorem", IconName: "change_history"},
	{Id: "derivatives", Title: "Derivatives", Description: "Measure how functions change at each point", LessonId: "intro_to_derivatives", IconName: "show_chart"},
	{Id: "linear", Title: "Linear Equations", Description: "Find the value of x in equations like 3x + 5 = 11", LessonId: "linear_equations", IconName: "linear_scale"},
}

// SnapOptions is the fixed set of actions offered after a Snap & Solve
// photo is submitted. Ported verbatim from MockTutiServerClient's
// _snapOptions; icon names are the Flutter Icons.* identifiers used there.
var SnapOptions = []*tutiv1.SnapOption{
	{Id: "check_work", Label: "Check my work", Description: "Find mistakes in my attempt and show me what went wrong.", IconName: "rate_review_outlined"},
	{Id: "solve", Label: "Solve this for me", Description: "Walk me through a step-by-step solution.", IconName: "auto_fix_high_outlined"},
	{Id: "explain", Label: "Explain the concept", Description: "Show me the underlying lesson so I can solve it myself.", IconName: "school_outlined"},
}

// lessonRefs mirrors each lesson's headline info for use as a
// LessonReference, keyed by lesson id. The linear/quadratic descriptions
// match MockTutiServerClient's _lessonsToReview verbatim; the rest are
// synthesized from the corresponding lesson's own content/icon since the
// mock never needed to reference them this way.
var lessonRefs = map[string]*tutiv1.LessonReference{
	"linear_equations": {
		LessonId: "linear_equations", Title: "Linear Equations",
		Description: "Solving equations of the form ax + b = c", IconName: "linear_scale",
	},
	"quadratic_equations": {
		LessonId: "quadratic_equations", Title: "Quadratic Equations",
		Description: "Factoring and the quadratic formula", IconName: "functions",
	},
	"pythagorean_theorem": {
		LessonId: "pythagorean_theorem", Title: "Pythagorean Theorem",
		Description: "Relating the sides of a right triangle", IconName: "change_history",
	},
	"intro_to_derivatives": {
		LessonId: "intro_to_derivatives", Title: "Introduction to Derivatives",
		Description: "Measuring how functions change at each point", IconName: "show_chart",
	},
	"statistics_mean_median": {
		LessonId: "statistics_mean_median", Title: "Mean & Median",
		Description: "Summarising a dataset with a single number", IconName: "bar_chart",
	},
}

// topicGroup ties a topic to the lessons and practice problems that go
// with it, used to deterministically fill lessonsToReview/similarProblems
// instead of asking the model to invent valid ids.
type topicGroup struct {
	lessonIDs  []string
	problemIDs []string
}

var topicGroups = map[string]topicGroup{
	"algebra":    {lessonIDs: []string{"linear_equations", "quadratic_equations"}, problemIDs: []string{"prob_linear_2", "prob_quad_1"}},
	"geometry":   {lessonIDs: []string{"pythagorean_theorem"}, problemIDs: []string{"prob_pyth_1"}},
	"calculus":   {lessonIDs: []string{"intro_to_derivatives"}, problemIDs: nil},
	"statistics": {lessonIDs: []string{"statistics_mean_median"}, problemIDs: nil},
}

// defaultTopicGroup is used when the extracted topic doesn't match any
// known group — matches the mock's default pairing (linear + quadratic).
var defaultTopicGroup = topicGroups["algebra"]

// LessonsAndProblemsForTopic deterministically resolves lessonsToReview
// and similarProblems for a given (model-extracted) topic string, falling
// back to the default group for anything unrecognized.
func LessonsAndProblemsForTopic(topic string) ([]*tutiv1.LessonReference, []*tutiv1.Problem) {
	group, ok := topicGroups[strings.ToLower(strings.TrimSpace(topic))]
	if !ok {
		group = defaultTopicGroup
	}

	lessons := make([]*tutiv1.LessonReference, 0, len(group.lessonIDs))
	for _, id := range group.lessonIDs {
		if ref, ok := lessonRefs[id]; ok {
			lessons = append(lessons, ref)
		}
	}

	problems := make([]*tutiv1.Problem, 0, len(group.problemIDs))
	for _, id := range group.problemIDs {
		if p, ok := Problems[id]; ok {
			problems = append(problems, p)
		}
	}

	return lessons, problems
}
