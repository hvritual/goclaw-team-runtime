// dddgen:service-implementation TodoService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type TodoService struct{}

func NewTodoService() *TodoService { return &TodoService{} }

func (s *TodoService) CreateTodo(ctx context.Context, request contract.CreateTodoRequest) (contract.CreateTodoResponse, error) {
	return contract.CreateTodoResponse{}, contract.ErrTodoNotImplemented
}

func (s *TodoService) UpdateTodoStatus(ctx context.Context, request contract.UpdateTodoStatusRequest) (contract.UpdateTodoStatusResponse, error) {
	return contract.UpdateTodoStatusResponse{}, contract.ErrTodoNotImplemented
}
func (s *TodoService) GetTodo(ctx context.Context, request contract.GetTodoRequest) (contract.GetTodoResponse, error) {
	return contract.GetTodoResponse{}, contract.ErrTodoNotImplemented
}
func (s *TodoService) ListTodos(ctx context.Context, request contract.ListTodosRequest) (contract.ListTodosResponse, error) {
	return contract.ListTodosResponse{}, contract.ErrTodoNotImplemented
}
func (s *TodoService) UpdateTodo(ctx context.Context, request contract.UpdateTodoRequest) (contract.UpdateTodoResponse, error) {
	return contract.UpdateTodoResponse{}, contract.ErrTodoNotImplemented
}
func (s *TodoService) DeleteTodo(ctx context.Context, request contract.DeleteTodoRequest) (contract.DeleteTodoResponse, error) {
	return contract.DeleteTodoResponse{}, contract.ErrTodoNotImplemented
}

func (s *TodoService) RestoreTodo(ctx context.Context, request contract.RestoreTodoRequest) (contract.RestoreTodoResponse, error) {
	return contract.RestoreTodoResponse{}, contract.ErrTodoNotImplemented
}

func (s *TodoService) ReorderTodos(ctx context.Context, request contract.ReorderTodosRequest) (contract.ReorderTodosResponse, error) {
	return contract.ReorderTodosResponse{}, contract.ErrTodoNotImplemented
}
