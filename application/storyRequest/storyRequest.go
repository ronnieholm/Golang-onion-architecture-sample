package storyRequest

// TODO: Rename package to story?

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/models"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/seedwork"
	dseedwork "github.com/ronnieholm/golang-onion-architecture-sample/domain/seedwork"
	"github.com/ronnieholm/golang-onion-architecture-sample/domain/story"
)

// CaptureBasicStoryDetailsCommand

type CaptureBasicStoryDetailsCommand struct {
	Id          uuid.UUID
	Title       string
	Description *string
}

type captureBasicStoryDetailsValidatedCommand struct {
	Id          story.StoryId
	Title       story.StoryTitle
	Description *story.StoryDescription
}

func (c CaptureBasicStoryDetailsCommand) validate() (*captureBasicStoryDetailsValidatedCommand, error) {
	errs := seedwork.NewValidationErrors()
	id, err := story.NewStoryId(c.Id)
	errs.Set("Id", err)
	title, err := story.NewStoryTitle(c.Title)
	errs.Set("Title", err)

	var description *story.StoryDescription = nil
	if c.Description != nil {
		description, err = story.NewStoryDescription(*c.Description)
		errs.Set("Description", err)
	}

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return &captureBasicStoryDetailsValidatedCommand{
		Id:          *id,
		Title:       *title,
		Description: description,
	}, nil
}

func (c CaptureBasicStoryDetailsCommand) Run(ctx context.Context, identity seedwork.ScrumIdentity, stories story.Store, clock seedwork.Clock) (*uuid.UUID, error) {
	if !identity.IsInRole(seedwork.ScrumRoleMember) {
		return nil, seedwork.AuthorizationError{Role: seedwork.ScrumRoleMember}
	}

	cmd, err := c.validate()
	if err != nil {
		return nil, err
	}

	storyExists := stories.Exists(ctx, cmd.Id)
	if storyExists {
		return nil, seedwork.EntityConflictError{Entity: "Story", Id: c.Id}
	}

	story, event := story.CaptureBasicStoryDetails(cmd.Id, cmd.Title, cmd.Description, clock.UtcNow())
	stories.ApplyEvent(ctx, *event)
	return &story.Id.Value, nil
}

// ReviseBasicStoryDetailsCommand

type ReviseBasicStoryDetailsCommand struct {
	Id          uuid.UUID
	Title       string
	Description *string
}

type reviseBasicStoryDetailsValidatedCommand struct {
	Id          story.StoryId
	Title       story.StoryTitle
	Description *story.StoryDescription
}

func (c ReviseBasicStoryDetailsCommand) validate() (*reviseBasicStoryDetailsValidatedCommand, error) {
	errs := seedwork.NewValidationErrors()
	id, err := story.NewStoryId(c.Id)
	errs.Set("Id", err)
	title, err := story.NewStoryTitle(c.Title)
	errs.Set("Title", err)

	var description *story.StoryDescription = nil
	if c.Description != nil {
		description, err = story.NewStoryDescription(*c.Description)
		errs.Set("Description", err)
	}

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return &reviseBasicStoryDetailsValidatedCommand{
		Id:          *id,
		Title:       *title,
		Description: description,
	}, nil
}

func (c ReviseBasicStoryDetailsCommand) Run(ctx context.Context, identity seedwork.ScrumIdentity, stories story.Store, clock seedwork.Clock) (*uuid.UUID, error) {
	if !identity.IsInRole(seedwork.ScrumRoleMember) {
		return nil, seedwork.AuthorizationError{Role: seedwork.ScrumRoleMember}
	}

	cmd, err := c.validate()
	if err != nil {
		return nil, err
	}

	story := stories.GetById(ctx, cmd.Id)
	if story == nil {
		return nil, seedwork.EntityNotFoundError{Entity: "Story", Id: c.Id}
	}

	event := story.ReviseBasicStoryDetails(cmd.Title, cmd.Description, clock.UtcNow())
	stories.ApplyEvent(ctx, event)
	return &story.Id.Value, nil
}

// RemoveStoryCommand

type RemoveStoryCommand struct{ Id uuid.UUID }
type removeStoryValidatedCommand struct{ Id story.StoryId }

func (c RemoveStoryCommand) validate() (*removeStoryValidatedCommand, error) {
	errs := seedwork.NewValidationErrors()
	id, err := story.NewStoryId(c.Id)
	errs.Set("Id", err)

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return &removeStoryValidatedCommand{
		Id: *id,
	}, nil
}

func (c RemoveStoryCommand) Run(ctx context.Context, identity seedwork.ScrumIdentity, stories story.Store, clock seedwork.Clock) (*uuid.UUID, error) {
	if !identity.IsInRole(seedwork.ScrumRoleMember) {
		return nil, seedwork.AuthorizationError{Role: seedwork.ScrumRoleMember}
	}

	cmd, err := c.validate()
	if err != nil {
		return nil, err
	}

	story := stories.GetById(ctx, cmd.Id)
	if story == nil {
		return nil, seedwork.EntityNotFoundError{Entity: "Story", Id: c.Id}
	}

	event := story.Remove(clock.UtcNow())
	stories.ApplyEvent(ctx, event)
	return &story.Id.Value, nil
}

// AddBasicTaskDetailsToStoryCommand

type AddBasicTaskDetailsToStoryCommand struct {
	StoryId     uuid.UUID
	TaskId      uuid.UUID
	Title       string
	Description *string
}

type addBasicTaskDetailsToStoryValidatedCommand struct {
	StoryId     story.StoryId
	TaskId      story.TaskId
	Title       story.TaskTitle
	Description *story.TaskDescription
}

func (c AddBasicTaskDetailsToStoryCommand) validate() (*addBasicTaskDetailsToStoryValidatedCommand, error) {
	errs := seedwork.NewValidationErrors()
	storyId, err := story.NewStoryId(c.StoryId)
	errs.Set("StoryId", err)
	taskId, err := story.NewTaskId(c.TaskId)
	errs.Set("TaskId", err)
	title, err := story.NewTaskTitle(c.Title)
	errs.Set("Title", err)

	var description *story.TaskDescription = nil
	if c.Description != nil {
		description, err = story.NewTaskDescription(*c.Description)
		errs.Set("Description", err)
	}

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return &addBasicTaskDetailsToStoryValidatedCommand{
		StoryId:     *storyId,
		TaskId:      *taskId,
		Title:       *title,
		Description: description,
	}, nil
}

func (c AddBasicTaskDetailsToStoryCommand) Run(ctx context.Context, identity seedwork.ScrumIdentity, stories story.Store, clock seedwork.Clock) (*uuid.UUID, error) {
	if !identity.IsInRole(seedwork.ScrumRoleMember) {
		return nil, seedwork.AuthorizationError{Role: seedwork.ScrumRoleMember}
	}

	cmd, err := c.validate()
	if err != nil {
		return nil, err
	}

	s := stories.GetById(ctx, cmd.StoryId)
	if s == nil {
		return nil, seedwork.EntityNotFoundError{Entity: "Story", Id: cmd.StoryId.Value}
	}

	event, err := s.AddBasicTaskDetailsToStory(cmd.TaskId, cmd.Title, cmd.Description, clock.UtcNow())
	if err != nil {
		if errors.Is(err, story.ErrDuplicateTask) {
			return nil, seedwork.EntityConflictError{Entity: "Task", Id: cmd.TaskId.Value}
		}
		panic(fmt.Sprintf("unreachable: %T", err))
	}

	stories.ApplyEvent(ctx, *event)
	return &cmd.TaskId.Value, nil
}

// ReviseBasicTaskDetailsCommand

type ReviseBasicTaskDetailsCommand struct {
	StoryId     uuid.UUID
	TaskId      uuid.UUID
	Title       string
	Description *string
}

type ReviseBasicTaskDetailsValidatedCommand struct {
	StoryId     story.StoryId
	TaskId      story.TaskId
	Title       story.TaskTitle
	Description *story.TaskDescription
}

func (c ReviseBasicTaskDetailsCommand) validate() (*ReviseBasicTaskDetailsValidatedCommand, error) {
	errs := seedwork.NewValidationErrors()
	storyId, err := story.NewStoryId(c.StoryId)
	errs.Set("StoryId", err)
	taskId, err := story.NewTaskId(c.TaskId)
	errs.Set("TaskId", err)
	title, err := story.NewTaskTitle(c.Title)
	errs.Set("Title", err)
	var description *story.TaskDescription = nil
	if c.Description != nil {
		description, err = story.NewTaskDescription(*c.Description)
		errs.Set("Description", err)
	}

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return &ReviseBasicTaskDetailsValidatedCommand{
		StoryId:     *storyId,
		TaskId:      *taskId,
		Title:       *title,
		Description: description,
	}, nil
}

func (c ReviseBasicTaskDetailsCommand) Run(ctx context.Context, identity seedwork.ScrumIdentity, stories story.Store, clock seedwork.Clock) (*uuid.UUID, error) {
	if !identity.IsInRole(seedwork.ScrumRoleMember) {
		return nil, seedwork.AuthorizationError{Role: seedwork.ScrumRoleMember}
	}

	cmd, err := c.validate()
	if err != nil {
		return nil, err
	}

	s := stories.GetById(ctx, cmd.StoryId)
	if s == nil {
		return nil, seedwork.EntityNotFoundError{Entity: "Story", Id: cmd.StoryId.Value}
	}

	event, err := s.ReviseBasicTaskDetails(cmd.TaskId, cmd.Title, cmd.Description, clock.UtcNow())
	if err != nil {
		if errors.Is(err, story.ErrTaskNotFound) {
			return nil, seedwork.EntityNotFoundError{Entity: "Task", Id: cmd.TaskId.Value}
		}
		panic(fmt.Sprintf("unreachable: %T", err))
	}

	stories.ApplyEvent(ctx, *event)
	return &cmd.TaskId.Value, nil
}

// RemoveTaskCommand

type RemoveTaskCommand struct {
	StoryId uuid.UUID
	TaskId  uuid.UUID
}

// TODO: *Validated can be unexported?

type RemoveTaskValidatedCommand struct {
	StoryId story.StoryId
	TaskId  story.TaskId
}

func (c RemoveTaskCommand) validate() (*RemoveTaskValidatedCommand, error) {
	errs := seedwork.NewValidationErrors()
	storyId, err := story.NewStoryId(c.StoryId)
	errs.Set("StoryId", err)
	taskId, err := story.NewTaskId(c.TaskId)
	errs.Set("TaskId", err)

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return &RemoveTaskValidatedCommand{
		StoryId: *storyId,
		TaskId:  *taskId,
	}, nil
}

func (c RemoveTaskCommand) Run(ctx context.Context, identity seedwork.ScrumIdentity, stories story.Store, clock seedwork.Clock) (*uuid.UUID, error) {
	if !identity.IsInRole(seedwork.ScrumRoleMember) {
		return nil, seedwork.AuthorizationError{Role: seedwork.ScrumRoleMember}
	}

	cmd, err := c.validate()
	if err != nil {
		return nil, err
	}

	s := stories.GetById(ctx, cmd.StoryId)
	if s == nil {
		return nil, seedwork.EntityNotFoundError{Entity: "Story", Id: cmd.StoryId.Value}
	}

	event, err := s.RemoveTask(cmd.TaskId, clock.UtcNow())
	if err != nil {
		if errors.Is(err, story.ErrTaskNotFound) {
			return nil, seedwork.EntityNotFoundError{Entity: "Task", Id: cmd.TaskId.Value}
		}
		panic(fmt.Sprintf("unreachable: %T", err))
	}

	stories.ApplyEvent(ctx, *event)
	return &cmd.TaskId.Value, nil
}

// GetByIdQuery

type GetStoryByIdQuery struct{ Id uuid.UUID }
type GetStoryByIdValidatedQuery struct{ Id story.StoryId }

type TaskDto struct {
	Id          uuid.UUID
	Title       string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

type StoryDto struct {
	Id          uuid.UUID
	Title       string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	Tasks       []TaskDto
}

func fromTask(t story.Task) TaskDto {
	var description *string
	if t.Description != nil {
		description = &t.Description.Value
	}

	return TaskDto{
		Id:          t.Id.Value,
		Title:       t.Title.Value,
		Description: description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func fromStory(s story.Story) StoryDto {
	var description *string
	if s.Description != nil {
		description = &s.Description.Value
	}

	tasks := make([]TaskDto, 0, len(s.Tasks))
	for _, t := range s.Tasks {
		tasks = append(tasks, fromTask(t))
	}

	return StoryDto{
		Id:          s.Id.Value,
		Title:       s.Title.Value,
		Description: description,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		Tasks:       tasks,
	}
}

func (q GetStoryByIdQuery) validate() (*GetStoryByIdValidatedQuery, error) {
	errs := seedwork.NewValidationErrors()
	id, err := story.NewStoryId(q.Id)
	errs.Set("Id", err)

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return &GetStoryByIdValidatedQuery{
		Id: *id,
	}, nil
}

func (q GetStoryByIdQuery) Run(ctx context.Context, identity seedwork.ScrumIdentity, stories story.Store) (*StoryDto, error) {
	if !identity.IsInRole(seedwork.ScrumRoleMember) {
		return nil, seedwork.AuthorizationError{Role: seedwork.ScrumRoleMember}
	}

	qry, err := q.validate()
	if err != nil {
		return nil, err
	}

	story := stories.GetById(ctx, qry.Id)
	if story == nil {
		return nil, seedwork.EntityNotFoundError{Entity: "Story", Id: q.Id}
	}

	dto := fromStory(*story)
	return &dto, nil
}

// GetStoriesPagedQuery

type GetStoriesPagedQuery struct {
	Limit  int
	Cursor *string
}

type GetStoriesPagedValidatedQuery struct {
	Limit  dseedwork.Limit
	Cursor *dseedwork.Cursor
}

func (q GetStoriesPagedQuery) validate() (*GetStoriesPagedValidatedQuery, error) {
	errs := seedwork.NewValidationErrors()
	limit, err := dseedwork.NewLimit(q.Limit)
	errs.Set("Limit", err)

	var cursor *dseedwork.Cursor = nil
	if q.Cursor != nil {
		cursor, err = dseedwork.NewCursor(*q.Cursor)
		errs.Set("Cursor", err)
	}

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return &GetStoriesPagedValidatedQuery{
		Limit:  *limit,
		Cursor: cursor,
	}, nil
}

func (q GetStoriesPagedQuery) Run(ctx context.Context, identity seedwork.ScrumIdentity, stories story.Store) (*models.PagedDto[StoryDto], error) {
	if !identity.IsInRole(seedwork.ScrumRoleMember) {
		return nil, seedwork.AuthorizationError{Role: seedwork.ScrumRoleMember}
	}

	qry, err := q.validate()
	if err != nil {
		return nil, err
	}

	storiesPage := stories.GetPaged(ctx, qry.Limit, qry.Cursor)
	storyDtos := make([]StoryDto, 0, len(storiesPage.Items))
	for _, story := range storiesPage.Items {
		storyDtos = append(storyDtos, fromStory(story))
	}

	var cursor *string = nil
	if storiesPage.Cursor != nil {
		cursor = &storiesPage.Cursor.Value
	}

	return &models.PagedDto[StoryDto]{
		Cursor: cursor,
		Items:  storyDtos,
	}, nil
}
