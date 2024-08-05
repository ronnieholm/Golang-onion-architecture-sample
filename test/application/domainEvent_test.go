package application

import (
	"context"
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/domainEvent"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/seedwork"
	"github.com/ronnieholm/golang-onion-architecture-sample/infrastructure"
	"github.com/ronnieholm/golang-onion-architecture-sample/infrastructure/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DomainEventTestSuite struct {
	suite.Suite
	tx      *sql.Tx
	clock   infrastructure.Clock
	stories sqlite.StoryStore
	events  sqlite.DomainEventStore
}

func (s *DomainEventTestSuite) SetupTest() {
	tx := reset()
	s.tx = tx
	s.clock = infrastructure.Clock{}
	s.stories = sqlite.StoryStore{Tx: tx}
	s.events = sqlite.DomainEventStore{Tx: tx}
}

func (s *DomainEventTestSuite) TearDownTest() {
	err := s.tx.Commit()
	if err != nil {
		panic(err)
	}
}

func (s *DomainEventTestSuite) TestMustHaveAdminRoleToQueryDomainEvents() {
	require := require.New(s.T())
	ctx := context.Background()
	_, err := domainEvent.GetByAggregateIdQuery{Id: missingId(), Limit: 5, Cursor: nil}.Run(ctx, memberIdentity, s.events)
	require.ErrorAs(err, &seedwork.ErrAuthorization)
	authErr := err.(seedwork.AuthorizationError)
	require.Equal(authErr.Role, seedwork.ScrumRoleAdmin)
}

// TODO: UserId -> Id

func (s *DomainEventTestSuite) TestGetByAggregateIdPaged() {
	require := require.New(s.T())
	ctx := context.Background()
	const (
		stories = 1
		tasks   = 14
		events  = stories + tasks
	)
	cmd := captureBasicStoryDetailsCommand()
	cmd.Run(ctx, memberIdentity, s.stories, s.clock)
	for i := 1; i <= tasks; i++ {
		cmd := addBasicTaskDetailsToStoryCommand(cmd.Id)
		cmd.Title = strconv.Itoa(i)
		cmd.Run(ctx, memberIdentity, s.stories, s.clock)
	}

	page1, err := domainEvent.GetByAggregateIdQuery{Id: cmd.Id, Limit: 5, Cursor: nil}.Run(ctx, adminIdentity, s.events)
	require.NoError(err)
	page2, err := domainEvent.GetByAggregateIdQuery{Id: cmd.Id, Limit: 5, Cursor: page1.Cursor}.Run(ctx, adminIdentity, s.events)
	require.NoError(err)
	page3, err := domainEvent.GetByAggregateIdQuery{Id: cmd.Id, Limit: 5, Cursor: page2.Cursor}.Run(ctx, adminIdentity, s.events)
	require.NoError(err)

	require.Equal(5, len(page1.Items))
	require.Equal(5, len(page2.Items))
	require.Equal(5, len(page3.Items))

	uniqueCreatedAt := make(map[time.Time]struct{}, events)
	uniqueAggregateIds := make(map[uuid.UUID]struct{}, stories)
	uniqueAggregateTypes := make(map[string]struct{}, stories)
	for _, s := range page1.Items {
		uniqueCreatedAt[s.CreatedAt] = struct{}{}
		uniqueAggregateIds[s.AggregateId] = struct{}{}
		uniqueAggregateTypes[s.AggregateType] = struct{}{}
	}
	for _, s := range page2.Items {
		uniqueCreatedAt[s.CreatedAt] = struct{}{}
		uniqueAggregateIds[s.AggregateId] = struct{}{}
		uniqueAggregateTypes[s.AggregateType] = struct{}{}
	}
	for _, s := range page3.Items {
		uniqueCreatedAt[s.CreatedAt] = struct{}{}
		uniqueAggregateIds[s.AggregateId] = struct{}{}
		uniqueAggregateTypes[s.AggregateType] = struct{}{}
	}

	require.Equal(events, len(uniqueCreatedAt))
	require.Equal(stories, len(uniqueAggregateIds))
	for k := range uniqueAggregateIds {
		require.Equal(cmd.Id, k)
	}
	require.Equal(stories, len(uniqueAggregateTypes))
	for k := range uniqueAggregateTypes {
		require.Equal("Story", k)
	}
}

func TestDomainEventTestSuite(t *testing.T) {
	suite.Run(t, new(DomainEventTestSuite))
}
