package orchestratorlite

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func allWorkItemIDs(task Task) []string {
	var ids []string
	for _, milestone := range task.Plan.Milestones {
		for _, item := range milestone.WorkItems {
			if strings.TrimSpace(item.ID) != "" {
				ids = append(ids, item.ID)
			}
		}
	}
	return uniqueStrings(ids)
}

func setAllWorkItemsStatus(task *Task, status WorkItemStatus) {
	for milestoneIndex := range task.Plan.Milestones {
		for itemIndex := range task.Plan.Milestones[milestoneIndex].WorkItems {
			task.Plan.Milestones[milestoneIndex].WorkItems[itemIndex].Status = status
		}
	}
}

func attributeChangedFiles(task Task, changedFiles []string) ([]ChangeAttribution, []string) {
	type indexedItem struct {
		id       string
		issueIDs []string
		paths    []string
	}
	var items []indexedItem
	for _, milestone := range task.Plan.Milestones {
		for _, item := range milestone.WorkItems {
			items = append(items, indexedItem{
				id:       item.ID,
				issueIDs: uniqueStrings(append(append([]string(nil), task.IssueIDs...), item.IssueIDs...)),
				paths:    item.ScopePaths,
			})
		}
	}

	attributed := make(map[string][]string)
	var unattributed []string
	for _, changedFile := range changedFiles {
		matches := 0
		for _, item := range items {
			if len(item.paths) > 0 && matchesAnyPath(changedFile, item.paths) {
				attributed[item.id] = append(attributed[item.id], changedFile)
				matches++
			}
		}
		if matches == 0 && len(items) == 1 {
			attributed[items[0].id] = append(attributed[items[0].id], changedFile)
			matches = 1
		}
		if matches == 0 {
			unattributed = append(unattributed, changedFile)
		}
	}

	var result []ChangeAttribution
	for _, item := range items {
		files := uniqueStrings(attributed[item.id])
		if len(files) == 0 {
			continue
		}
		result = append(result, ChangeAttribution{
			WorkItemID: item.id,
			IssueIDs:   item.issueIDs,
			Files:      files,
		})
	}
	return result, uniqueStrings(unattributed)
}

func commitMessageWithTraceability(task Task, message string) string {
	message = strings.TrimSpace(message)
	trailers := []string{
		"Task-ID: " + task.ID,
		"Project-ID: " + task.ProjectID,
		"Task-Revision: " + fmt.Sprintf("%d", task.Compile.Revision),
	}
	if task.RepositoryID != "" {
		trailers = append(trailers, "Repository-ID: "+task.RepositoryID)
	}
	if task.CorrelationID != "" {
		trailers = append(trailers, "Correlation-ID: "+task.CorrelationID)
	}
	if task.PolicyBundleHash != "" {
		trailers = append(trailers, "Policy-Bundle: "+task.PolicyBundleHash)
	}
	if task.Wave != nil {
		trailers = append(
			trailers,
			"Wave-ID: "+task.Wave.WaveID,
			"Wave-Revision: "+strconv.Itoa(task.Wave.PlanRevision),
			"Wave-Step: "+task.Wave.StepID,
		)
	}
	for _, workItemID := range allWorkItemIDs(task) {
		trailers = append(trailers, "Work-Item: "+workItemID)
	}
	for _, issueID := range uniqueStrings(task.IssueIDs) {
		trailers = append(trailers, "Issue: "+issueID)
	}
	return message + "\n\n" + strings.Join(trailers, "\n")
}

// RecordPullRequest links a committed task to its code-review surface. It does
// not create or merge the pull request; provider adapters own those mutations.
func (s *Service) RecordPullRequest(id, actor, rawURL string) (Task, error) {
	release, err := s.acquireRun(id, false)
	if err != nil {
		return Task{}, err
	}
	defer release()

	task, err := s.GetTask(id)
	if err != nil {
		return Task{}, err
	}
	if task.CommitSHA == "" {
		return Task{}, errors.New("task must be committed before a pull request can be linked")
	}
	pullRequestURL, err := parsePullRequestURL(rawURL)
	if err != nil {
		return Task{}, err
	}
	if task.PullRequestURL != "" {
		if task.PullRequestURL == pullRequestURL {
			return task, nil
		}
		return Task{}, fmt.Errorf(
			"task is already linked to pull request %s",
			task.PullRequestURL,
		)
	}
	task.PullRequestURL = pullRequestURL
	task.UpdatedAt = time.Now().UTC()
	if err := s.recordTaskEvent(task, "task.pull_request_linked", valueOr(actor, "human"), map[string]any{
		"pull_request_url": task.PullRequestURL,
		"commit_sha":       task.CommitSHA,
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

// RecordImportedPullRequest binds a commit created outside the control plane
// back to an accepted Workstation patch. The commit must be available in the
// registered local repository, descend from the frozen base, reproduce the
// accepted diff exactly (apart from Git's index metadata), and carry all
// traceability trailers. This method records identity only; it never fetches,
// pushes, creates, approves, or merges a pull request.
func (s *Service) RecordImportedPullRequest(
	ctx context.Context,
	id, actor, rawCommit, rawURL string,
) (Task, error) {
	release, err := s.acquireRun(id, false)
	if err != nil {
		return Task{}, err
	}
	defer release()

	task, err := s.GetTask(id)
	if err != nil {
		return Task{}, err
	}
	if task.Status != TaskDone {
		return Task{}, fmt.Errorf(
			"task %s must be accepted before an external commit can be linked",
			id,
		)
	}
	imported, err := verifyImportedEvidenceTree(task)
	if err != nil {
		return Task{}, err
	}
	if !imported {
		return Task{}, errors.New(
			"external commit linking is only valid for imported workstation evidence",
		)
	}
	pullRequestURL, err := parsePullRequestURL(rawURL)
	if err != nil {
		return Task{}, err
	}
	commitRef := strings.ToLower(strings.TrimSpace(rawCommit))
	if !validGitCommitReference(commitRef) {
		return Task{}, errors.New(
			"commit_sha must be a 7-64 character hexadecimal Git commit id",
		)
	}
	resolved, err := runGit(
		ctx,
		task.RepoPath,
		"rev-parse",
		"--verify",
		commitRef+"^{commit}",
	)
	if err != nil {
		return Task{}, fmt.Errorf(
			"resolve external commit %q in registered repository: %w: %s",
			commitRef,
			err,
			strings.TrimSpace(resolved.Stderr),
		)
	}
	commitSHA := strings.ToLower(strings.TrimSpace(resolved.Stdout))
	if task.CommitSHA != "" {
		if task.CommitSHA == commitSHA && task.PullRequestURL == pullRequestURL {
			return task, nil
		}
		return Task{}, fmt.Errorf(
			"task is already linked to commit %s and pull request %s",
			task.CommitSHA,
			task.PullRequestURL,
		)
	}
	if ancestor, ancestorErr := runGit(
		ctx,
		task.RepoPath,
		"merge-base",
		"--is-ancestor",
		task.Compile.BaseCommit,
		commitSHA,
	); ancestorErr != nil {
		return Task{}, fmt.Errorf(
			"external commit %s does not descend from frozen base %s: %w: %s",
			commitSHA,
			task.Compile.BaseCommit,
			ancestorErr,
			strings.TrimSpace(ancestor.Stderr),
		)
	}
	commitDiff, err := runGit(
		ctx,
		task.RepoPath,
		"diff",
		"--binary",
		"--no-ext-diff",
		task.Compile.BaseCommit,
		commitSHA,
		"--",
	)
	if err != nil {
		return Task{}, fmt.Errorf(
			"read external commit diff: %w: %s",
			err,
			strings.TrimSpace(commitDiff.Stderr),
		)
	}
	acceptedDiff, err := os.ReadFile(filepath.Join(task.LastEvidence, "diff.patch"))
	if err != nil {
		return Task{}, fmt.Errorf("read accepted imported diff: %w", err)
	}
	if canonicalCommitLinkDiff(commitDiff.Stdout) !=
		canonicalCommitLinkDiff(string(acceptedDiff)) {
		return Task{}, errors.New(
			"external commit diff does not exactly match the accepted workstation patch",
		)
	}
	message, err := runGit(
		ctx,
		task.RepoPath,
		"show",
		"-s",
		"--format=%B",
		commitSHA,
	)
	if err != nil {
		return Task{}, fmt.Errorf(
			"read external commit message: %w: %s",
			err,
			strings.TrimSpace(message.Stderr),
		)
	}
	if missing := missingCommitTrailers(task, message.Stdout); len(missing) > 0 {
		return Task{}, fmt.Errorf(
			"external commit is missing traceability trailers: %s",
			strings.Join(missing, ", "),
		)
	}

	task.CommitSHA = commitSHA
	task.PullRequestURL = pullRequestURL
	task.UpdatedAt = time.Now().UTC()
	if err := s.recordTaskEvent(
		task,
		"task.external_pull_request_linked",
		valueOr(actor, "human"),
		map[string]any{
			"pull_request_url":          task.PullRequestURL,
			"commit_sha":                task.CommitSHA,
			"base_commit":               task.Compile.BaseCommit,
			"patch_sha256":              sha256Bytes(acceptedDiff),
			"commit_verified":           true,
			"pull_request_url_verified": false,
		},
	); err != nil {
		return Task{}, err
	}
	return task, nil
}

func parsePullRequestURL(rawURL string) (string, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") ||
		parsed.Host == "" ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return "", errors.New(
			"pull request URL must be an absolute http(s) URL without credentials, query parameters, or fragments",
		)
	}
	return parsed.String(), nil
}

func validGitCommitReference(value string) bool {
	if len(value) < 7 || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func canonicalCommitLinkDiff(diff string) string {
	var result strings.Builder
	for _, line := range strings.SplitAfter(diff, "\n") {
		withoutNewline := strings.TrimSuffix(line, "\n")
		withoutNewline = strings.TrimSuffix(withoutNewline, "\r")
		if strings.HasPrefix(withoutNewline, "index ") {
			continue
		}
		result.WriteString(line)
	}
	return result.String()
}

func missingCommitTrailers(task Task, message string) []string {
	required := []string{
		"Task-ID: " + task.ID,
		"Project-ID: " + task.ProjectID,
		"Task-Revision: " + strconv.Itoa(task.Compile.Revision),
	}
	if task.RepositoryID != "" {
		required = append(required, "Repository-ID: "+task.RepositoryID)
	}
	if task.CorrelationID != "" {
		required = append(required, "Correlation-ID: "+task.CorrelationID)
	}
	if task.PolicyBundleHash != "" {
		required = append(required, "Policy-Bundle: "+task.PolicyBundleHash)
	}
	if task.Wave != nil {
		required = append(
			required,
			"Wave-ID: "+task.Wave.WaveID,
			"Wave-Revision: "+strconv.Itoa(task.Wave.PlanRevision),
			"Wave-Step: "+task.Wave.StepID,
		)
	}
	for _, workItemID := range allWorkItemIDs(task) {
		required = append(required, "Work-Item: "+workItemID)
	}
	for _, issueID := range uniqueStrings(task.IssueIDs) {
		required = append(required, "Issue: "+issueID)
	}
	lines := make(map[string]struct{})
	for _, line := range strings.Split(message, "\n") {
		lines[strings.TrimSpace(line)] = struct{}{}
	}
	var missing []string
	for _, trailer := range required {
		if _, ok := lines[trailer]; !ok {
			missing = append(missing, trailer)
		}
	}
	return missing
}
