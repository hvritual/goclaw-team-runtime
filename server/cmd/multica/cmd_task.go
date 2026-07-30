package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage workspace tasks",
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspace tasks",
	RunE:  runTaskList,
}

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a task",
	RunE:  runTaskCreate,
}

var taskUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a task",
	Args:  exactArgs(1),
	RunE:  runTaskUpdate,
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Args:  exactArgs(1),
	RunE:  runTaskDelete,
}

func init() {
	taskCmd.AddCommand(taskListCmd, taskCreateCmd, taskUpdateCmd, taskDeleteCmd)

	taskListCmd.Flags().String("status", "", "Filter by status")
	taskListCmd.Flags().String("output", "table", "Output format: table or json")

	taskCreateCmd.Flags().String("title", "", "Task title (required)")
	taskCreateCmd.Flags().String("description", "", "Task description")
	taskCreateCmd.Flags().String("status", "todo", "Task status")
	taskCreateCmd.Flags().String("priority", "none", "Task priority")

	taskUpdateCmd.Flags().String("title", "", "New task title")
	taskUpdateCmd.Flags().String("description", "", "New task description")
	taskUpdateCmd.Flags().String("status", "", "New task status")
	taskUpdateCmd.Flags().String("priority", "", "New task priority")
}

func runTaskList(cmd *cobra.Command, _ []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	params := url.Values{}
	if status, _ := cmd.Flags().GetString("status"); status != "" {
		params.Set("status", status)
	}
	path := "/api/tasks"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	var response struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := client.GetJSON(ctx, path, &response); err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}
	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, response.Tasks)
	}
	rows := make([][]string, 0, len(response.Tasks))
	for _, task := range response.Tasks {
		rows = append(rows, []string{
			displayID(strVal(task, "id"), false),
			strVal(task, "title"),
			strVal(task, "status"),
			strVal(task, "priority"),
		})
	}
	cli.PrintTable(os.Stdout, []string{"ID", "TITLE", "STATUS", "PRIORITY"}, rows)
	return nil
}

func runTaskCreate(cmd *cobra.Command, _ []string) error {
	title, _ := cmd.Flags().GetString("title")
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("--title is required")
	}
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	body := map[string]any{
		"title":    title,
		"status":   flagString(cmd, "status"),
		"priority": flagString(cmd, "priority"),
	}
	if description := flagString(cmd, "description"); description != "" {
		body["description"] = description
	}
	var task map[string]any
	if err := client.PostJSON(ctx, "/api/tasks", body, &task); err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return cli.PrintJSON(os.Stdout, task)
}

func runTaskUpdate(cmd *cobra.Command, args []string) error {
	body := map[string]any{}
	for _, field := range []string{"title", "description", "status", "priority"} {
		if cmd.Flags().Changed(field) {
			body[field] = flagString(cmd, field)
		}
	}
	if len(body) == 0 {
		return fmt.Errorf("at least one update flag is required")
	}
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	var task map[string]any
	if err := client.PutJSON(ctx, "/api/tasks/"+url.PathEscape(args[0]), body, &task); err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return cli.PrintJSON(os.Stdout, task)
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	if err := client.DeleteJSON(ctx, "/api/tasks/"+url.PathEscape(args[0])); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	fmt.Fprintln(os.Stdout, "Task deleted.")
	return nil
}

func flagString(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return strings.TrimSpace(value)
}
