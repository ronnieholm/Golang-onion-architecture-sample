package story

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ronnieholm/golang-onion-architecture-sample/domain/seedwork"
	"github.com/ronnieholm/golang-onion-architecture-sample/domain/validation"
)

// Task entity

type TaskId struct {
	Value uuid.UUID
}

func NewTaskId(value uuid.UUID) (*TaskId, error) {
	if err := validation.UuidNotEmpty(value); err != nil {
		return nil, err
	}
	return &TaskId{value}, nil
}

type TaskTitle struct {
	Value string
}

func NewTaskTitle(value string) (*TaskTitle, error) {
	if err := validation.StringNotNilOrWhitespace(value); err != nil {
		return nil, err
	}
	if err := validation.StringMaxLength(100, value); err != nil {
		return nil, err
	}
	return &TaskTitle{value}, nil
}

type TaskDescription struct {
	Value string
}

func NewTaskDescription(value string) (*TaskDescription, error) {
	if err := validation.StringNotNilOrWhitespace(value); err != nil {
		return nil, err
	}
	if err := validation.StringMaxLength(1000, value); err != nil {
		return nil, err
	}
	return &TaskDescription{value}, nil
}

type Task struct {
	seedwork.Entity[TaskId]
	Title       TaskTitle
	Description *TaskDescription
}

func NewTask(id TaskId, title TaskTitle, description *TaskDescription, createdAt time.Time) Task {
	return Task{
		Entity:      seedwork.Entity[TaskId]{Id: id, CreatedAt: createdAt, UpdatedAt: nil},
		Title:       title,
		Description: description,
	}
}

func (t Task) Equal(other Task) bool {
	return t.Entity.Id == other.Entity.Id
}

// Story root entity

var (
	ErrDuplicateTask = errors.New("duplicate task")
	ErrTaskNotFound  = errors.New("task not found")
)

type StoryId struct {
	Value uuid.UUID
}

func NewStoryId(value uuid.UUID) (*StoryId, error) {
	if err := validation.UuidNotEmpty(value); err != nil {
		return nil, err
	}
	return &StoryId{value}, nil
}

type StoryTitle struct {
	Value string
}

func NewStoryTitle(value string) (*StoryTitle, error) {
	if err := validation.StringNotNilOrWhitespace(value); err != nil {
		return nil, err
	}
	if err := validation.StringMaxLength(100, value); err != nil {
		return nil, err
	}
	return &StoryTitle{value}, nil
}

type StoryDescription struct {
	Value string
}

func NewStoryDescription(value string) (*StoryDescription, error) {
	if err := validation.StringNotNilOrWhitespace(value); err != nil {
		return nil, err
	}
	if err := validation.StringMaxLength(1000, value); err != nil {
		return nil, err
	}
	return &StoryDescription{value}, nil
}

type Story struct {
	seedwork.AggregateRoot[StoryId]
	Title       StoryTitle
	Description *StoryDescription
	Tasks       []Task
}

type BasicStoryDetailsCaptured struct {
	seedwork.DomainEvent
	StoryId          StoryId
	StoryTitle       StoryTitle
	StoryDescription *StoryDescription
}

type BasicStoryDetailsRevised struct {
	seedwork.DomainEvent
	StoryId          StoryId
	StoryTitle       StoryTitle
	StoryDescription *StoryDescription
}

type StoryRemoved struct {
	seedwork.DomainEvent
	StoryId StoryId
}

type BasicTaskDetailsAddedToStory struct {
	seedwork.DomainEvent
	StoryId         StoryId
	TaskId          TaskId
	TaskTitle       TaskTitle
	TaskDescription *TaskDescription
}

type BasicTaskDetailsRevised struct {
	seedwork.DomainEvent
	StoryId         StoryId
	TaskId          TaskId
	TaskTitle       TaskTitle
	TaskDescription *TaskDescription
}

type TaskRemoved struct {
	seedwork.DomainEvent
	StoryId StoryId
	TaskId  TaskId
}

func CaptureBasicStoryDetails(id StoryId, title StoryTitle, description *StoryDescription, createdAt time.Time) (*Story, *BasicStoryDetailsCaptured) {
	return &Story{
			AggregateRoot: seedwork.AggregateRoot[StoryId]{Id: id, CreatedAt: createdAt, UpdatedAt: nil},
			Title:         title,
			Description:   description,
			Tasks:         []Task{},
		},
		&BasicStoryDetailsCaptured{
			DomainEvent:      seedwork.DomainEvent{OccurredAt: createdAt},
			StoryId:          id,
			StoryTitle:       title,
			StoryDescription: description,
		}
}

func (s *Story) ReviseBasicStoryDetails(title StoryTitle, description *StoryDescription, updatedAt time.Time) BasicStoryDetailsRevised {
	s.UpdatedAt = &updatedAt
	s.Title = title
	s.Description = description
	return BasicStoryDetailsRevised{
		DomainEvent:      seedwork.DomainEvent{OccurredAt: updatedAt},
		StoryId:          s.Id,
		StoryTitle:       s.Title,
		StoryDescription: s.Description,
	}
}

func (s *Story) Remove(occurredAt time.Time) StoryRemoved {
	return StoryRemoved{
		DomainEvent: seedwork.DomainEvent{OccurredAt: occurredAt},
		StoryId:     s.Id,
	}
}

func (s *Story) AddBasicTaskDetailsToStory(taskId TaskId, title TaskTitle, description *TaskDescription, createdAt time.Time) (*BasicTaskDetailsAddedToStory, error) {
	task := NewTask(taskId, title, description, createdAt)

	duplicate := false
	for _, t := range s.Tasks {
		if t.Equal(task) {
			duplicate = true
			break
		}
	}
	if duplicate {
		return nil, ErrDuplicateTask
	}

	s.Tasks = append(s.Tasks, task)
	return &BasicTaskDetailsAddedToStory{
		DomainEvent:     seedwork.DomainEvent{OccurredAt: createdAt},
		StoryId:         s.Id,
		TaskId:          task.Id,
		TaskTitle:       task.Title,
		TaskDescription: task.Description,
	}, nil
}

func (s *Story) ReviseBasicTaskDetails(taskId TaskId, title TaskTitle, description *TaskDescription, updatedAt time.Time) (*BasicTaskDetailsRevised, error) {
	var task *Task = nil
	for _, t := range s.Tasks {
		if t.Id.Value == taskId.Value {
			task = &t
			break
		}
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}

	task.Title = title
	task.Description = description
	task.UpdatedAt = &updatedAt
	return &BasicTaskDetailsRevised{
		DomainEvent:     seedwork.DomainEvent{OccurredAt: updatedAt},
		StoryId:         s.Id,
		TaskId:          taskId,
		TaskTitle:       title,
		TaskDescription: description,
	}, nil
}

func (s *Story) RemoveTask(taskId TaskId, occuredAt time.Time) (*TaskRemoved, error) {
	foundIdx := -1
	for idx, t := range s.Tasks {
		if t.Id.Value == taskId.Value {
			foundIdx = idx
			break
		}
	}
	if foundIdx == -1 {
		return nil, ErrTaskNotFound
	}

	s.Tasks = append(s.Tasks[:foundIdx], s.Tasks[foundIdx+1:]...)
	return &TaskRemoved{
		DomainEvent: seedwork.DomainEvent{OccurredAt: occuredAt},
		StoryId:     s.Id,
		TaskId:      taskId,
	}, nil
}

type Store interface {
	Exists(ctx context.Context, id StoryId) bool
	GetById(ctx context.Context, id StoryId) *Story
	GetPaged(ctx context.Context, limit seedwork.Limit, cursor *seedwork.Cursor) seedwork.Paged[Story]
	ApplyEvent(ctx context.Context, event any)
}
