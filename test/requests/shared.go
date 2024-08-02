package requests

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/seedwork"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/story"
)

// TODO: rename to admin and member? Rename UserId to ID?

var adminIdentity = seedwork.ScrumIdentityAuthenticated{UserId: "123", Roles: []seedwork.ScrumRole{seedwork.ScrumRoleAdmin}}
var memberIdentity = seedwork.ScrumIdentityAuthenticated{UserId: "123", Roles: []seedwork.ScrumRole{seedwork.ScrumRoleMember}}

func captureBasicStoryDetailsCommand() story.CaptureBasicStoryDetailsCommand {
	description := "description"
	return story.CaptureBasicStoryDetailsCommand{
		Id:          uuid.New(),
		Title:       "title",
		Description: &description,
	}
}

func reviseBasicStoryDetailsCommand(storyId uuid.UUID) story.ReviseBasicStoryDetailsCommand {
	description := "description1"
	return story.ReviseBasicStoryDetailsCommand{
		Id:          storyId,
		Title:       "title1",
		Description: &description,
	}
}

func addBasicTaskDetailsToStoryCommand(storyId uuid.UUID) story.AddBasicTaskDetailsToStoryCommand {
	description := "description"
	return story.AddBasicTaskDetailsToStoryCommand{
		StoryId:     storyId,
		TaskId:      uuid.New(),
		Title:       "title",
		Description: &description,
	}
}

func reviseBasicTaskDetailsCommand(storyId, taskId uuid.UUID) story.ReviseBasicTaskDetailsCommand {
	description := "description1"
	return story.ReviseBasicTaskDetailsCommand{
		StoryId:     storyId,
		TaskId:      taskId,
		Title:       "title1",
		Description: &description,
	}
}

func missingId() uuid.UUID {
	return uuid.New()
}

func reset() *sql.Tx {
	db, err := sql.Open("sqlite3", "/home/rh/git/Golang-onion-architecture-sample/scrum_test.sqlite")
	if err != nil { // TODO: Add assert?
		panic(err)
	}

	sql := []string{
		"delete from tasks where id like '%'",
		"delete from stories where id like '%'",
		"delete from domain_events where id like '%'",
	}

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	for _, s := range sql {
		_, err := tx.Exec(s)
		if err != nil {
			panic(err)
		}
	}

	err = tx.Commit()
	if err != nil {
		panic(err)
	}

	tx, err = db.Begin()
	if err != nil {
		panic(err)
	}
	return tx
}
