package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrInvalidProjectName = errors.New("project name is required and cannot be empty")
	ErrInvalidProjectID   = errors.New("project id is required")
)

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateProjectInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (in *CreateProjectInput) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return ErrInvalidProjectName
	}
	if len(in.Name) > 255 {
		return errors.New("project name must be at most 255 characters")
	}
	return nil
}

type UpdateProjectInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (in *UpdateProjectInput) Validate() error {
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return ErrInvalidProjectName
		}
		if len(trimmed) > 255 {
			return errors.New("project name must be at most 255 characters")
		}
		*in.Name = trimmed
	}
	return nil
}
