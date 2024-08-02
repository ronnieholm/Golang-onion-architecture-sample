package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
		log.Printf("Did error happen?")
		next.ServeHTTP(w, r)
	}
}

func requestLoggerMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("method %s, path %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func requireAuthMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Is user authenticated?")
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
	_ []string,
	_ func(string) string,
	_ io.Reader,
	_, _ io.Writer,
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
		port: "8080",
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
		_ = context.Background() // TODO: why doesn't this work from the blog post?
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()

	wg.Wait()
	return nil
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

	// mux.HandleFunc("/healthz", handleHealthzPlease(logger))
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
	Id string `json:"id"`
}

func handleStoryCreate(db *sql.DB, clock *infrastructure.Clock, storyStoreCreator func(*sql.Tx) *sqlite.StoryStore) http.Handler {
	// TODO: temporary. Call function with request to parse identity.
	identity := seedwork.ScrumIdentityAuthenticated{
		UserId: "123",
		Roles:  []seedwork.ScrumRole{seedwork.ScrumRoleMember}}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx, err := db.Begin()
		if err != nil {
			panic(err) // TODO: don't panic
		}
		stories := storyStoreCreator(tx)

		req, err := decode[StoryCreateRequest](r)
		if err != nil {
			fmt.Println(err.Error())
			return // TODO: Return an http response
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
			err1 := tx.Rollback()
			if err1 != nil {
				// TODO: what to do except log there error? We can't return it.
				fmt.Println(err1)
			}
			return // TODO: Return an http response
		}

		err = tx.Commit()
		if err != nil {
			// TODO: what to do?
			fmt.Println(err)
		}

		err = encode(w, r, http.StatusOK, StoryCreateResponse{Id: id.String()})
		if err != nil {
			fmt.Println(err.Error())
			return // TODO: Return an http response
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
	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
