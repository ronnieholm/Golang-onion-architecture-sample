package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/seedwork"
	"github.com/ronnieholm/golang-onion-architecture-sample/application/story"
	"github.com/ronnieholm/golang-onion-architecture-sample/infrastructure"
	"github.com/ronnieholm/golang-onion-architecture-sample/infrastructure/sqlite"
)

func NewServer(
	/* logger, */
	db *sql.DB,
	clock *infrastructure.Clock,
	storyStoreCreator func(*sql.Tx) *sqlite.StoryStore,
	domainEventStore *sqlite.DomainEventStore,
) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, db, clock, storyStoreCreator, domainEventStore)

	var handler http.Handler = mux
	handler = globalErrorMiddleware( /* logger, */ handler)
	handler = requestLoggerMiddleware( /* logger, */ handler)
	handler = requireAuthMiddleware( /* logger, */ handler)
	return handler
}

func globalErrorMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//log.Printf("Did error happen?")
		next.ServeHTTP(w, r)
	}
}

func requestLoggerMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//log.Printf("method %s, path %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func requireAuthMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//log.Printf("Is user authenticated?")
		_ = r.Header.Get("Authorization")
		next.ServeHTTP(w, r)
	}
}

func setupDatabase() (*sql.DB, func(), error) {
	db, err := sql.Open("sqlite3", "/home/rh/git/Golang-onion-architecture-sample/scrum_web.sqlite")
	if err != nil {
		return nil, nil, err
	}
	tidy := func() {
		err = db.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to close database: %s", err)
		}
	}
	return db, tidy, nil
}

type config struct {
	host, port string
}

// Pass in operating system fundaments, such as os.Args if the service has flag
// support or maybe even os.Stdin, os.Stdout, os.Stderr.
func run(
	ctx context.Context,
	// _ []string,
	// _ func(string) string,
	// _ io.Reader,
	// _, _ io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	db, cleanup, err := setupDatabase()
	if err != nil {
		return fmt.Errorf("setup database: %w", err)
	}
	defer cleanup()

	config := config{
		host: "localhost",
		port: "5000",
	}

	srv := NewServer(
		/* logger, */
		/* config, */
		db,
		&infrastructure.Clock{},
		func(tx *sql.Tx) *sqlite.StoryStore { return &sqlite.StoryStore{Tx: tx} },
		&sqlite.DomainEventStore{},
	)

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.host, config.port),
		Handler: srv,
	}

	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		// make a new context for the Shutdown.
		//shutdownCtx := context.Background() // TODO: why doesn't this work from the blog post?
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()

	wg.Wait()
	return nil
}

type ProblemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

type ValidationErrorDto struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func inferContentType(r *http.Request) string {
	contentType := "application/json"
outer:
	for k, v := range r.Header {
		if k == "acceptHeaders" {
			for _, v1 := range v {
				if v1 == "application/problem+json" {
					contentType = v1
					break outer
				}
			}
		}
	}
	return contentType
}

func newProblemDetails(status int, detail string) ProblemDetails {
	return ProblemDetails{
		Type:   "Error",
		Title:  "Error",
		Status: status,
		Detail: detail,
	}
}

func x(w http.ResponseWriter, r *http.Request, pd ProblemDetails) {
	w.Header().Set("Content-Type", inferContentType(r))
	w.WriteHeader(pd.Status)
	if err := json.NewEncoder(w).Encode(pd); err != nil {
		panic(err)
	}
}

func missingQueryStringParam(name string) {

}

func unexpectedQueryStringParam(name string) {

}

func queryStringParamMustBeOfType(name, type_ string) {

}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var pd ProblemDetails

	// TODO: slog here for each case as well?

	switch err := err.(type) {
	case seedwork.AuthorizationError:
		pd = newProblemDetails(http.StatusUnauthorized, fmt.Sprintf("Missing role: {%s}", err.Role))
	case seedwork.ValidationErrors:
		errs := make([]ValidationErrorDto, 0, len(err.Errors))
		for _, e := range err.Errors {
			errs = append(errs, ValidationErrorDto{Field: e.Field, Message: e.Message})
		}

		errsJson, err1 := json.MarshalIndent(errs, "", "  ")
		if err1 != nil {
			panic(err)
		}
		pd = newProblemDetails(http.StatusBadRequest, string(errsJson))
	case seedwork.EntityConflictError:
		pd = newProblemDetails(http.StatusConflict, "EntityConflictError")
	case seedwork.EntityNotFoundError:
		pd = newProblemDetails(http.StatusNotFound, "EntityNotFoundError")
	case seedwork.ApplicationError:
		pd = newProblemDetails(http.StatusInternalServerError, "ApplicationError")
	case TxBeginError:
		pd = newProblemDetails(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	case RequestDecodeError:
		pd = newProblemDetails(http.StatusBadRequest, err.Error())
	case TxCommitError:
		pd = newProblemDetails(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	default:
		pd = newProblemDetails(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	x(w, r, pd)
}

func getCurrentIdentity(_ *http.Request) seedwork.ScrumIdentityAuthenticated {
	// TODO: parse request
	return seedwork.ScrumIdentityAuthenticated{
		UserId: "123",
		Roles:  []seedwork.ScrumRole{seedwork.ScrumRoleMember}}
}

// TODO: should arguments be interface? I believe so for mockability.
func addRoutes(
	mux *http.ServeMux,
	db *sql.DB,
	clock *infrastructure.Clock,
	storyStoreCreator func(*sql.Tx) *sqlite.StoryStore,
	domainEventStore *sqlite.DomainEventStore,
) {
	// TODO: How to only apply middleware at select routes? Move routes from other function?

	mux.Handle("POST /authentication/issue-token", handleIssueToken())
	mux.Handle("POST /authentication/renew-token", handleRenewToken())
	mux.Handle("POST /authentication/introspect", handleIntrospectToken())

	mux.Handle("POST /stories", handleStoryCreate(db, clock, storyStoreCreator))
	mux.Handle("PUT /stories/{id}", handleStoryUpdate())
	mux.Handle("POST /stories/{id}/tasks", handleTaskCreate())
	mux.Handle("PUT /stories/{storyId}/tasks/{taskId}", handleTaskUpdate())
	mux.Handle("DELETE /stories/{storyId}/tasks/{taskId}", handleTaskRemove())
	mux.Handle("DELETE /stories/{id}", handleStoryRemove())
	mux.Handle("GET /stories/{id}", handleStoryGet())
	mux.Handle("GET /stories", handleStoryPaged())

	mux.Handle("GET /persisted-domain-events/{id}", handleDomainEventsPaged(db, domainEventStore))

	mux.Handle("/health", handleHealth())
	// mux.Handle("/", http.NotFoundHandler())
}

func handleIssueToken() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

func handleRenewToken() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

func handleIntrospectToken() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

type StoryCreateRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
}

type StoryCreateResponse struct {
	StoryId string `json:"story_id"`
}

type TxBeginError struct {
	Err error
}

func (e TxBeginError) Error() string {
	return "TODO: TxBeginError"
}

type RequestDecodeError struct {
	Err error
}

func (e RequestDecodeError) Error() string {
	return "TODO: RequestDecodeError"
}

type TxCommitError struct {
	Err error
}

func (e TxCommitError) Error() string {
	return "TODO: TxCommitError"
}

func handleHealth() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})
}

// TODO: Consider reducing boilerplate by implementing https://stackoverflow.com/questions/16184238/database-sql-tx-detecting-commit-or-rollback

func handleStoryCreate(db *sql.DB, clock *infrastructure.Clock, storyStoreCreator func(*sql.Tx) *sqlite.StoryStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx, err := db.Begin()
		if err != nil {
			writeError(w, r, TxBeginError{err})
			return
		}
		defer func() {
			err := tx.Rollback()
			if err != nil && err.Error() != "sql: transaction has already been committed or rolled back" {
				fmt.Printf("todo: log %s", err)
			}
		}()

		stories := storyStoreCreator(tx)
		identity := getCurrentIdentity(r)

		req, err := decode[StoryCreateRequest](r)
		if err != nil {
			writeError(w, r, RequestDecodeError{err})
			return
		}

		storyId, err := uuid.NewUUID()
		if err != nil {
			panic(err)
		}
		cmd := story.CaptureBasicStoryDetailsCommand{
			Id:          storyId,
			Title:       req.Title,
			Description: req.Description,
		}
		id, err := cmd.Run(r.Context(), identity, stories, clock)
		if err != nil {
			writeError(w, r, err)
			return
		}

		err = tx.Commit()
		if err != nil {
			writeError(w, r, TxCommitError{err})
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/stories/%s", id.String()))
		err = encode(w, r, http.StatusCreated, StoryCreateResponse{StoryId: id.String()})
		if err != nil {
			panic(err)
		}
	})
}

func handleStoryUpdate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

func handleTaskCreate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

func handleTaskUpdate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

func handleTaskRemove() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

func handleStoryRemove() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

func handleStoryGet() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

func handleStoryPaged() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

func handleDomainEventsPaged(_ *sql.DB, _ *sqlite.DomainEventStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Write([]byte("Id: " + id))
	})
}

// Are encode and decode worth it?

func encode[T any](w http.ResponseWriter, _ *http.Request, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func decode[T any](r *http.Request) (T, error) {
	defer r.Body.Close() // TODO: is this required?

	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx /*os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr*/); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
