# Reseller Loyalty

Create a `.env` file at the root of the repository with the following content:

    DB_BASE_URL=postgres://postgres:secret@localhost:5432
    DB_LOCAL_NAME=reseller_loyalty_local
    DB_LOCAL_INTEGRATION_TEST_NAME=reseller_loyalty_local_integration_test  

Then

    $ docker compose up

starts up a PostgreSQL server at `localhost:5432` and PgAdmin at
http://localhost:5500 (see `docker-compose.yml` for login details).

Assuming Go is installed locally, next

    $ make db-create
    $ make db-migrate-up
    $ make build
    $ make test
    $ make db-drop # optional

## Migrations

Create a new migration:

    $ make migrate-create