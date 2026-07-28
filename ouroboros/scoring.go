package ouroboros

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type clarityInput struct {
	Goal          float64
	GoalWhy       string
	Constraint    float64
	ConstraintWhy string
	Success       float64
	SuccessWhy    string
	Context       float64
	ContextWhy    string
}

func calculateAmbiguity(
	input clarityInput,
	brownfield bool,
	threshold float64,
	round int,
	readyStreak int,
	requiredReadyStreak int,
	summary string,
	unresolved []string,
) (AmbiguityAssessment, error) {
	if threshold <= 0 || threshold >= 1 {
		return AmbiguityAssessment{}, fmt.Errorf("ambiguity threshold must be between 0 and 1")
	}
	values := []struct {
		dimension Dimension
		clarity   float64
		weight    float64
		floor     float64
		why       string
	}{
		{DimensionGoal, input.Goal, 0.40, 0.75, input.GoalWhy},
		{DimensionConstraint, input.Constraint, 0.30, 0.65, input.ConstraintWhy},
		{DimensionSuccess, input.Success, 0.30, 0.70, input.SuccessWhy},
	}
	if brownfield {
		values = []struct {
			dimension Dimension
			clarity   float64
			weight    float64
			floor     float64
			why       string
		}{
			{DimensionGoal, input.Goal, 0.35, 0.75, input.GoalWhy},
			{DimensionConstraint, input.Constraint, 0.25, 0.65, input.ConstraintWhy},
			{DimensionSuccess, input.Success, 0.25, 0.70, input.SuccessWhy},
			{DimensionContext, input.Context, 0.15, 0.60, input.ContextWhy},
		}
	}

	clarity := 0.0
	floorsPassed := true
	dimensions := make([]DimensionScore, 0, len(values))
	for _, value := range values {
		if value.clarity < 0 || value.clarity > 1 || math.IsNaN(value.clarity) || math.IsInf(value.clarity, 0) {
			return AmbiguityAssessment{}, fmt.Errorf("%s clarity must be between 0 and 1", value.dimension)
		}
		weighted := value.clarity * value.weight
		clarity += weighted
		floorPassed := value.clarity >= value.floor
		floorsPassed = floorsPassed && floorPassed
		dimensions = append(dimensions, DimensionScore{
			Dimension:     value.dimension,
			Clarity:       round4(value.clarity),
			Weight:        value.weight,
			WeightedValue: round4(weighted),
			Justification: strings.TrimSpace(value.why),
			Floor:         value.floor,
			FloorPassed:   floorPassed,
		})
	}
	overall := round4(math.Max(0, math.Min(1, 1-clarity)))
	qualifies := overall <= threshold && floorsPassed
	if qualifies {
		readyStreak++
	} else {
		readyStreak = 0
	}
	if requiredReadyStreak < 1 {
		requiredReadyStreak = 1
	}
	return AmbiguityAssessment{
		Round:               round,
		Overall:             overall,
		Threshold:           threshold,
		Dimensions:          dimensions,
		Ready:               qualifies && readyStreak >= requiredReadyStreak,
		ReadyStreak:         readyStreak,
		RequiredReadyStreak: requiredReadyStreak,
		Summary:             strings.TrimSpace(summary),
		Unresolved:          cleanStrings(unresolved),
	}, nil
}

func ontologySimilarity(left, right Ontology) float64 {
	leftFields := ontologyFieldMap(left.Fields)
	rightFields := ontologyFieldMap(right.Fields)
	if len(leftFields) == 0 && len(rightFields) == 0 {
		if normalizeText(left.Name) == normalizeText(right.Name) &&
			normalizeText(left.Description) == normalizeText(right.Description) {
			return 1
		}
		return 0
	}

	union := make(map[string]struct{}, len(leftFields)+len(rightFields))
	for name := range leftFields {
		union[name] = struct{}{}
	}
	for name := range rightFields {
		union[name] = struct{}{}
	}
	shared := 0
	typeMatches := 0
	exactMatches := 0
	for name := range union {
		l, leftOK := leftFields[name]
		r, rightOK := rightFields[name]
		if !leftOK || !rightOK {
			continue
		}
		shared++
		if normalizeText(l.Type) == normalizeText(r.Type) {
			typeMatches++
		}
		if normalizeText(l.Type) == normalizeText(r.Type) &&
			normalizeText(l.Description) == normalizeText(r.Description) &&
			l.Required == r.Required {
			exactMatches++
		}
	}
	unionCount := len(union)
	nameOverlap := float64(shared) / float64(unionCount)
	typeMatch := 0.0
	exactMatch := 0.0
	if shared > 0 {
		typeMatch = float64(typeMatches) / float64(shared)
		exactMatch = float64(exactMatches) / float64(shared)
	}
	return round4(0.5*nameOverlap + 0.3*typeMatch + 0.2*exactMatch)
}

func ontologyFieldMap(fields []OntologyField) map[string]OntologyField {
	result := make(map[string]OntologyField, len(fields))
	for _, field := range fields {
		name := normalizeText(field.Name)
		if name == "" {
			continue
		}
		result[name] = field
	}
	return result
}

func questionOverlap(current, previous []Question) float64 {
	if len(current) == 0 || len(previous) == 0 {
		return 0
	}
	left := make(map[string]struct{}, len(current))
	right := make(map[string]struct{}, len(previous))
	for _, question := range current {
		for _, token := range significantTokens(question.Text) {
			left[token] = struct{}{}
		}
	}
	for _, question := range previous {
		for _, token := range significantTokens(question.Text) {
			right[token] = struct{}{}
		}
	}
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	intersection := 0
	union := make(map[string]struct{}, len(left)+len(right))
	for token := range left {
		union[token] = struct{}{}
		if _, ok := right[token]; ok {
			intersection++
		}
	}
	for token := range right {
		union[token] = struct{}{}
	}
	return round4(float64(intersection) / float64(len(union)))
}

var tokenPattern = regexp.MustCompile(`[\p{L}\p{N}_-]+`)

func significantTokens(value string) []string {
	matches := tokenPattern.FindAllString(strings.ToLower(value), -1)
	seen := make(map[string]struct{}, len(matches))
	var result []string
	for _, match := range matches {
		match = strings.TrimSpace(match)
		if len([]rune(match)) < 2 {
			continue
		}
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		result = append(result, match)
	}
	sort.Strings(result)
	return result
}

func normalizeText(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func cleanStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
