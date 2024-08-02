package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	dseedwork "github.com/ronnieholm/golang-onion-architecture-sample/domain/seedwork"
	"github.com/ronnieholm/golang-onion-architecture-sample/domain/story"
	"github.com/ronnieholm/golang-onion-architecture-sample/infrastructure"
)

type StoryStore struct {
	Tx *sql.Tx // TODO: is it allowed to copy the pointer? Otherwise use pointer receivers like for Mutex types
}

func (r StoryStore) Exists(ctx context.Context, id story.StoryId) bool {
	const sql = "select count(*) from stories where id = ?"
	res := r.Tx.QueryRowContext(ctx, sql, id.Value)
	count := -1
	err := res.Scan(&count)
	if err != nil {
		panic(fmt.Errorf("eixsts failed: %w", err))
	}
	return count == 1
}

func (r StoryStore) storiesToDomain(rows []getByIdRow) []story.Story {
	storyTasks := make(map[story.StoryId]map[story.TaskId]story.Task)
	storyTasksOrder := make(map[story.StoryId][]story.TaskId, 0)
	visitTask := func(storyId story.StoryId, r getByIdRow) {
		if r.tid != nil {
			taskIdRaw, err := uuid.Parse(*r.tid)
			if err != nil {
				panic(fmt.Errorf("unable to parse taskId %s as UUID: %w", *r.tid, err))
			}
			taskId, err := story.NewTaskId(taskIdRaw)
			if err != nil {
				panic(fmt.Errorf("invalid TaskId: %w", err))
			}
			task := r.parseTask(*taskId)

			tasks, storyVisited := storyTasks[storyId]
			if !storyVisited {
				tasks := make(map[story.TaskId]story.Task)
				tasks[*taskId] = task
				storyTasks[storyId] = tasks

				order := make([]story.TaskId, 0)
				order = append(order, *taskId)
				storyTasksOrder[storyId] = order
			} else {
				if _, taskVisited := tasks[*taskId]; !taskVisited {
					tasks[*taskId] = task
					order := storyTasksOrder[storyId]
					storyTasksOrder[storyId] = append(order, *taskId)
				}
			}
		}
	}

	stories := make(map[story.StoryId]story.Story)
	storiesOrder := make([]story.StoryId, 0)
	visitStory := func(r getByIdRow) {
		idRaw, err := uuid.Parse(r.sid)
		if err != nil {
			panic(fmt.Errorf("unable to parse storyId %s as UUID: %w", r.sid, err))
		}
		id, err := story.NewStoryId(idRaw)
		if err != nil {
			panic(fmt.Errorf("invalid storyId %v: %w", idRaw, err))
		}

		if _, ok := stories[*id]; !ok {
			story := r.parseStory(*id)
			stories[story.Id] = story
			storiesOrder = append(storiesOrder, *id)
		}
		visitTask(*id, r)
	}

	for _, r := range rows {
		visitStory(r)
	}

	if len(stories) != len(storiesOrder) {
		panic(fmt.Sprintf("length of stories %d and storiesOrder %d must be identitical", len(stories), len(storiesOrder)))
	}
	if len(storyTasks) != len(storyTasksOrder) {
		panic(fmt.Sprintf("length of storyTasks %d and storyTasksOrder %d must be identical", len(storyTasks), len(storyTasksOrder)))
	}

	result := make([]story.Story, 0, len(storiesOrder))
	for _, storyId := range storiesOrder {
		taskOrder, ok := storyTasksOrder[storyId]
		var tasks []story.Task // TODO: nil or empty slice? Initialization in domain object for inspiration? Should be consistent.
		if ok {
			tasks = make([]story.Task, 0, len(taskOrder))
			for _, taskId := range taskOrder {
				task := storyTasks[storyId][taskId]
				tasks = append(tasks, task)
			}
		}
		story := stories[storyId]
		story.Tasks = tasks
		result = append(result, story)
	}
	return result
}

type getByIdRow struct {
	sid          string
	stitle       string
	sdescription *string
	screatedat   int64
	supdatedat   *int64
	tid          *string
	ttitle       *string
	tdescription *string
	tcreatedat   *int64
	tupdatedat   *int64
}

func newGetByIdRow(res *sql.Rows) []getByIdRow {
	rows := make([]getByIdRow, 0)
	for res.Next() {
		r := getByIdRow{}
		err := res.Scan(&r.sid, &r.stitle, &r.sdescription, &r.screatedat, &r.supdatedat, &r.tid, &r.ttitle, &r.tdescription, &r.tcreatedat, &r.tupdatedat)
		if err != nil {
			panic(fmt.Errorf("getById scan failed: %w", err))
		}
		rows = append(rows, r)
	}
	return rows
}

func (r getByIdRow) parseTask(id story.TaskId) story.Task {
	title, err := story.NewTaskTitle(r.stitle)
	if err != nil {
		panic(fmt.Errorf("invalid TaskTitle %s: %w", r.stitle, err))
	}

	var description *story.TaskDescription = nil
	if r.sdescription != nil {
		description, err = story.NewTaskDescription(*r.sdescription)
		if err != nil {
			panic(fmt.Errorf("invalid TaskDescription %s: %w", *r.sdescription, err))
		}
	}

	createdAt := parseCreatedAt(r.screatedat)
	updatedAt := parseUpdatedAt(r.supdatedat)

	return story.Task{
		Entity:      dseedwork.Entity[story.TaskId]{Id: id, CreatedAt: createdAt, UpdatedAt: updatedAt},
		Title:       *title,
		Description: description,
	}
}

func (r getByIdRow) parseStory(id story.StoryId) story.Story {
	title, err := story.NewStoryTitle(r.stitle)
	if err != nil {
		panic(fmt.Errorf("invalid StoryTitle %s: %w", r.stitle, err))
	}

	var description *story.StoryDescription = nil
	if r.sdescription != nil {
		description, err = story.NewStoryDescription(*r.sdescription)
		if err != nil {
			panic(fmt.Errorf("invalid StoryDescription %s: %w", *r.sdescription, err))
		}
	}

	createdAt := parseCreatedAt(r.screatedat)
	updatedAt := parseUpdatedAt(r.supdatedat)

	return story.Story{
		AggregateRoot: dseedwork.AggregateRoot[story.StoryId]{Id: id, CreatedAt: createdAt, UpdatedAt: updatedAt},
		Title:         *title,
		Description:   description,
		Tasks:         []story.Task{},
	}
}

func (r StoryStore) GetById(ctx context.Context, id story.StoryId) *story.Story {
	const sql = `
		select s.id, s.title, s.description, s.created_at, s.updated_at,
               t.id, t.title, t.description, t.created_at, t.updated_at
		from stories s
		left join tasks t on s.id = t.story_id
		where s.id = ?`
	res, err := r.Tx.QueryContext(ctx, sql, id.Value)
	if err != nil {
		panic(fmt.Errorf("GetbyId query failed: %w", err))
	}

	rows := newGetByIdRow(res)
	stories := r.storiesToDomain(rows)
	switch len(stories) {
	case 0:
		return nil
	case 1:
		s := stories[0]
		return &s
	default:
		panic(fmt.Sprintf("Invalid database. %d instances with StoryId: %s", len(stories), id.Value.String()))
	}
}

func (r StoryStore) GetPaged(ctx context.Context, limit dseedwork.Limit, cursor *dseedwork.Cursor) dseedwork.Paged[story.Story] {
	const sql = `
		select s.id, s.title, s.description, s.created_at, s.updated_at,
               t.id, t.title, t.description, t.created_at, t.updated_at
		from stories s
		left join tasks t on s.id = t.story_id
		where s.created_at > ?
		order by s.created_at
		limit ?`

	offset, err := infrastructure.CursorToOffset(cursor)
	if err != nil {
		panic(fmt.Errorf("invalid cursor %s: %w", cursor.Value, err))
	}

	res, err := r.Tx.QueryContext(ctx, sql, offset, limit.Value)
	if err != nil {
		panic(fmt.Errorf("query failed: %w", err))
	}

	rows := newGetByIdRow(res)
	stories := r.storiesToDomain(rows)

	if len(stories) == 0 {
		return dseedwork.Paged[story.Story]{
			Cursor: nil,
			Items:  stories,
		}
	}

	pageEndOffset := stories[len(stories)-1].CreatedAt.UnixNano()
	globalEndOffset := getLargestCreatedAt("stories", r.Tx)
	newCursor := infrastructure.OffsetsToCursor(pageEndOffset, globalEndOffset)
	return dseedwork.Paged[story.Story]{
		Cursor: newCursor,
		Items:  stories,
	}
}

func (r StoryStore) ApplyEvent(ctx context.Context, event any) {
	var (
		aggregateId uuid.UUID
		occuredAt   time.Time
		res         sql.Result
		err         error
	)

	switch e := event.(type) {
	case story.BasicStoryDetailsCaptured:
		const sql = "insert into stories (id, title, description, created_at) values (?, ?, ?, ?)"
		var description *string = nil
		if e.StoryDescription != nil {
			description = &e.StoryDescription.Value
		}
		res, err = r.Tx.ExecContext(ctx, sql, e.StoryId.Value.String(), e.StoryTitle.Value, description, e.OccurredAt.UnixNano())
		aggregateId, occuredAt = e.StoryId.Value, e.OccurredAt
	case story.BasicStoryDetailsRevised:
		const sql = "update stories set title = ?, description = ?, updated_at = ? where id = ?"
		var description *string = nil
		if e.StoryDescription != nil {
			description = &e.StoryDescription.Value
		}
		res, err = r.Tx.ExecContext(ctx, sql, e.StoryTitle.Value, description, e.OccurredAt.UnixNano(), e.StoryId.Value.String())
		aggregateId, occuredAt = e.StoryId.Value, e.OccurredAt
	case story.StoryRemoved:
		const sql = "delete from stories where id = ?"
		res, err = r.Tx.ExecContext(ctx, sql, e.StoryId.Value.String())
		aggregateId, occuredAt = e.StoryId.Value, e.OccurredAt
	case story.BasicTaskDetailsAddedToStory:
		const sql = "insert into tasks (id, story_id, title, description, created_at) values (?, ?, ?, ?, ?)"
		var description *string = nil
		if e.TaskDescription != nil {
			description = &e.TaskDescription.Value
		}
		res, err = r.Tx.ExecContext(ctx, sql, e.TaskId.Value.String(), e.StoryId.Value.String(), e.TaskTitle.Value, description, e.OccurredAt.UnixNano())
		aggregateId, occuredAt = e.StoryId.Value, e.OccurredAt
	case story.BasicTaskDetailsRevised:
		const sql = "update tasks set title = ?, description = ?, updated_at = ? where id = ? and story_id = ?"
		var description *string = nil
		if e.TaskDescription != nil {
			description = &e.TaskDescription.Value
		}
		res, err = r.Tx.ExecContext(ctx, sql, e.TaskTitle.Value, description, e.OccurredAt.UnixNano(), e.TaskId.Value.String(), e.StoryId.Value.String())
		aggregateId, occuredAt = e.StoryId.Value, e.OccurredAt
	case story.TaskRemoved:
		const sql = "delete from tasks where id = ? and story_id = ?"
		res, err = r.Tx.ExecContext(ctx, sql, e.TaskId.Value.String(), e.StoryId.Value.String())
		aggregateId, occuredAt = e.StoryId.Value, e.OccurredAt
	default:
		panic(fmt.Sprintf("unreachable: %T", e))
	}

	if err != nil {
		panic(fmt.Errorf("failed to apply event %v: %w", event, err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		panic(fmt.Errorf("unable to get rows affected: %w", err))
	}
	if n != 1 {
		panic(fmt.Sprintf("rows affected expected 1, was %d", n))
	}
	if aggregateId == uuid.Nil {
		panic("aggregateId not set")
	}
	if occuredAt.IsZero() {
		panic("occurredAt not set")
	}

	persistDomainEvent(r.Tx, "Story", aggregateId, fmt.Sprintf("%T", event), fmt.Sprintf("%#v", event), occuredAt)
}
