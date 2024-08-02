package requests

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	"github.com/ronnieholm/golang-onion-architecture-sample/application/seedwork"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/story"
	"github.com/ronnieholm/golang-onion-architecture-sample/infrastructure"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type StoryTestSuite struct {
	suite.Suite
	tx      *sql.Tx
	clock   infrastructure.Clock
	stories infrastructure.SqlStoryStore
}

func (s *StoryTestSuite) SetupTest() {
	tx := reset()
	s.tx = tx
	s.clock = infrastructure.Clock{}
	s.stories = infrastructure.SqlStoryStore{Tx: tx}
}

func (s *StoryTestSuite) TearDownTest() {
	err := s.tx.Commit()
	if err != nil {
		panic(err)
	}
}

func (s *StoryTestSuite) TestMustHaveMemberRoleToCaptureBasicStoryDetails() {
	require := require.New(s.T())
	ctx := context.Background()
	cmd := captureBasicStoryDetailsCommand()
	_, err := cmd.Run(ctx, adminIdentity, s.stories, s.clock)
	require.ErrorAs(err, &seedwork.ErrAuthorization)
	authErr := err.(seedwork.AuthorizationError)
	require.Equal(authErr.Role, seedwork.ScrumRoleMember)
}

// TODO: F# and Go impl. doesn't assert createdAt as well as they could.

func (s *StoryTestSuite) TestCaptureBasicStoryAndTaskDetails() {
	require := require.New(s.T())
	ctx := context.Background()
	storyCmd := captureBasicStoryDetailsCommand()
	_, err := storyCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	taskCmd := addBasicTaskDetailsToStoryCommand(storyCmd.Id)
	_, err = taskCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	storyQry := story.GetStoryByIdQuery{Id: taskCmd.StoryId}
	actual, err := storyQry.Run(ctx, memberIdentity, s.stories)
	require.NoError(err)

	expected := story.StoryDto{
		Id:          storyCmd.Id,
		Title:       storyCmd.Title,
		Description: storyCmd.Description,
		CreatedAt:   actual.CreatedAt,
		UpdatedAt:   nil,
		Tasks: []story.TaskDto{
			{
				Id:          taskCmd.TaskId,
				Title:       taskCmd.Title,
				Description: taskCmd.Description,
				CreatedAt:   actual.Tasks[0].CreatedAt,
				UpdatedAt:   nil,
			},
		},
	}
	require.Equal(expected, *actual)
}

func (s *StoryTestSuite) TestCaptureDuplicateStory() {
	require := require.New(s.T())
	ctx := context.Background()
	cmd := captureBasicStoryDetailsCommand()
	_, err := cmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	_, err = cmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.ErrorAs(err, &seedwork.ErrEntityConflict)
	entErr := err.(seedwork.EntityConflictError)
	require.Equal("Story", entErr.Entity)
	require.Equal(cmd.Id, entErr.Id)
}

func (s *StoryTestSuite) TestRemoveStoryWithTask() {
	require := require.New(s.T())
	ctx := context.Background()
	storyCmd := captureBasicStoryDetailsCommand()
	_, err := storyCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	taskCmd := addBasicTaskDetailsToStoryCommand(storyCmd.Id)
	_, err = taskCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	removeCmd := story.RemoveStoryCommand{Id: storyCmd.Id}
	_, err = removeCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	storyQry := story.GetStoryByIdQuery{Id: taskCmd.StoryId}
	_, err = storyQry.Run(ctx, memberIdentity, s.stories)
	require.ErrorAs(err, &seedwork.EntityNotFoundError{})
	entErr := err.(seedwork.EntityNotFoundError)
	require.Equal(seedwork.EntityNotFoundError{Entity: "Story", Id: storyCmd.Id}, entErr)
}

func (s *StoryTestSuite) TestAddDuplicateTaskToStory() {
	require := require.New(s.T())
	ctx := context.Background()
	storyCmd := captureBasicStoryDetailsCommand()
	_, err := storyCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	taskCmd := addBasicTaskDetailsToStoryCommand(storyCmd.Id)
	_, err = taskCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	_, err = taskCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.ErrorAs(err, &seedwork.ErrEntityConflict)
	entErr := err.(seedwork.EntityConflictError)
	require.Equal(seedwork.EntityConflictError{Entity: "Task", Id: taskCmd.TaskId}, entErr)
}

func (s *StoryTestSuite) TestAddTaskToNonExistingStory() {
	require := require.New(s.T())
	ctx := context.Background()
	cmd := addBasicTaskDetailsToStoryCommand(missingId())
	_, err := cmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.ErrorAs(err, &seedwork.ErrEntityNotFound)
	entErr := err.(seedwork.EntityNotFoundError)
	require.Equal(seedwork.EntityNotFoundError{Entity: "Story", Id: cmd.StoryId}, entErr)
}

func (s *StoryTestSuite) TestRemoveTaskFromStory() {
	require := require.New(s.T())
	ctx := context.Background()
	storyCmd := captureBasicStoryDetailsCommand()
	_, err := storyCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	taskCmd := addBasicTaskDetailsToStoryCommand(storyCmd.Id)
	_, err = taskCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	removeCmd := story.RemoveTaskCommand{StoryId: taskCmd.StoryId, TaskId: taskCmd.TaskId}
	_, err = removeCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	// TODO: F#: Get the story and validate the task is gone.
}

func (s *StoryTestSuite) TestRemoveTaskFromNonExistingStory() {
	require := require.New(s.T())
	ctx := context.Background()
	cmd := story.RemoveTaskCommand{StoryId: missingId(), TaskId: missingId()}
	_, err := cmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.ErrorAs(err, &seedwork.ErrEntityNotFound)
	entErr := err.(seedwork.EntityNotFoundError)
	require.Equal(seedwork.EntityNotFoundError{Entity: "Story", Id: cmd.StoryId}, entErr)
}

func (s *StoryTestSuite) TestRemoveNonExistingTaskFromStory() {
	require := require.New(s.T())
	ctx := context.Background()
	storyCmd := captureBasicStoryDetailsCommand()
	_, err := storyCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	removeCmd := story.RemoveTaskCommand{StoryId: storyCmd.Id, TaskId: missingId()}
	_, err = removeCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.ErrorAs(err, &seedwork.EntityNotFoundError{})
	entErr := err.(seedwork.EntityNotFoundError)
	require.Equal(seedwork.EntityNotFoundError{Entity: "Task", Id: removeCmd.TaskId}, entErr)
}

func (s *StoryTestSuite) TestReviseBasicStoryDetails() {
	require := require.New(s.T())
	ctx := context.Background()
	storyCmd := captureBasicStoryDetailsCommand()
	_, err := storyCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	reviseCmd := reviseBasicStoryDetailsCommand(storyCmd.Id)
	_, err = reviseCmd.Run(ctx, memberIdentity, s.stories, s.clock)

	// TODO: F#: Actually validate fields after getting
	require.NoError(err)
}

func (s *StoryTestSuite) TestReviseNonExistingStory() {
	require := require.New(s.T())
	ctx := context.Background()
	cmd := reviseBasicStoryDetailsCommand(missingId())
	_, err := cmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.ErrorAs(err, &seedwork.EntityNotFoundError{})
	entErr := err.(seedwork.EntityNotFoundError)
	require.Equal(seedwork.EntityNotFoundError{Entity: "Story", Id: cmd.Id}, entErr)
}

func (s *StoryTestSuite) TestReviseBasicTaskDetails() {
	require := require.New(s.T())
	ctx := context.Background()
	storyCmd := captureBasicStoryDetailsCommand()
	_, err := storyCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	taskCmd := addBasicTaskDetailsToStoryCommand(storyCmd.Id)
	_, err = taskCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	reviseCmd := reviseBasicTaskDetailsCommand(taskCmd.StoryId, taskCmd.TaskId)
	_, err = reviseCmd.Run(ctx, memberIdentity, s.stories, s.clock)

	// TODO: F#: Actually validate fields after getting
	require.NoError(err)
}

func (s *StoryTestSuite) TestReviseNonExistingTask() {
	require := require.New(s.T())
	ctx := context.Background()
	storyCmd := captureBasicStoryDetailsCommand()
	_, err := storyCmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.NoError(err)

	reviseCmd := reviseBasicTaskDetailsCommand(storyCmd.Id, missingId())
	_, err = reviseCmd.Run(ctx, memberIdentity, s.stories, s.clock)

	require.ErrorAs(err, &seedwork.ErrEntityNotFound)
	entErr := err.(seedwork.EntityNotFoundError)
	require.Equal(seedwork.EntityNotFoundError{Entity: "Task", Id: reviseCmd.TaskId}, entErr)
}

func (s *StoryTestSuite) TestReviseTaskOnNonExistingStory() {
	require := require.New(s.T())
	ctx := context.Background()
	cmd := reviseBasicTaskDetailsCommand(missingId(), missingId())
	_, err := cmd.Run(ctx, memberIdentity, s.stories, s.clock)
	require.ErrorAs(err, &seedwork.ErrEntityNotFound)
	entErr := err.(seedwork.EntityNotFoundError)
	require.Equal(seedwork.EntityNotFoundError{Entity: "Story", Id: cmd.StoryId}, entErr)
}

func (s *StoryTestSuite) TestGetStoriesPaged() {
	require := require.New(s.T())
	ctx := context.Background()
	const stories = 14
	for i := 1; i <= stories; i++ {
		cmd := captureBasicStoryDetailsCommand()
		cmd.Title = strconv.Itoa(i)
		cmd.Run(ctx, memberIdentity, s.stories, s.clock)
	}

	page1, err := story.GetStoriesPagedQuery{Limit: 5, Cursor: nil}.Run(ctx, memberIdentity, s.stories)
	require.NoError(err)
	page2, err := story.GetStoriesPagedQuery{Limit: 5, Cursor: page1.Cursor}.Run(ctx, memberIdentity, s.stories)
	require.NoError(err)
	page3, err := story.GetStoriesPagedQuery{Limit: 5, Cursor: page2.Cursor}.Run(ctx, memberIdentity, s.stories)
	require.NoError(err)

	require.Equal(5, len(page1.Items))
	require.Equal(5, len(page2.Items))
	require.Equal(4, len(page3.Items))

	unique := make(map[string]bool, stories)
	for _, s := range page1.Items {
		unique[s.Title] = true
	}
	for _, s := range page2.Items {
		unique[s.Title] = true
	}
	for _, s := range page3.Items {
		unique[s.Title] = true
	}
	require.Equal(stories, len(unique))
}

func TestStoryTestSuite(t *testing.T) {
	suite.Run(t, new(StoryTestSuite))
}
