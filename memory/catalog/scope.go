package catalog

import (
	"context"
	"fmt"
	"strings"
)

type projectScopeKey struct{}

// WithProjectScope binds agent tools to the project selected by channel/topic
// routing. Tool parameters may narrow only to the same project.
func WithProjectScope(ctx context.Context, projectID string) context.Context {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ctx
	}
	return context.WithValue(ctx, projectScopeKey{}, projectID)
}

func ProjectScope(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(projectScopeKey{}).(string)
	return strings.TrimSpace(value)
}

func ResolveScopedProject(ctx context.Context, requested string) (string, error) {
	scoped := ProjectScope(ctx)
	requested = strings.TrimSpace(requested)
	if scoped == "" {
		return requested, nil
	}
	if requested != "" && requested != scoped {
		return "", fmt.Errorf(
			"project %q is outside the current run scope %q",
			requested,
			scoped,
		)
	}
	return scoped, nil
}
