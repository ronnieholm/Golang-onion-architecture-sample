# Golang onion architecture sample: Reseller Loyalty

_Got a comment or a question? Don't hesitate to drop me an email or open an issue._

This sample uses Go to develop a service in the domain driven desing/onion
architecture style. It stays true to the spirit of Go while not simplifying the
essential requirements of the service. As such, the sample limits the use of
external dependencies and in favor of custom solutions to request validation,
dependency injection, and object-relation mapping.

<img src="./docs/onion-architecture.png" width="550px" />

The sample is a modular monolith to offer the simplicity of a monolith and the
scalability of microservices. Conceptually each aggregate is a microservice, and
in-process an aggregate may interact with other aggregates through a
well-defined interface. Thus complexity increases sub-linearly with the number
of aggregates.

It includes the following features:

- Command Query Responsibility Segregation (CQRS) access to the application layer.
- Integration tests with the ability to fake any dependency.
- Integration tests use property based testing to supplement example tests.
- Database migrations and initial seeding.

## Getting started

Create a `.env` file at the root of the repository with the following content:

    DB_BASE_URL=postgres://postgres:secret@localhost:5432
    DB_LOCAL_NAME=reseller_loyalty_local
    DB_LOCAL_INTEGRATION_TEST_NAME=reseller_loyalty_local_integration_test  

Then run

    $ docker compose up

to run a PostgreSQL server at `localhost:5432` and a pgAdmin interface at
http://localhost:5500 (see `docker-compose.yml` for login details).

Assuming Make and Go are already installed, next

    $ make db-create
    $ make db-migrate-up
    $ make build
    $ make test

## Constraints

Not every project requires an implementation of every concept from domain driven
design and onion architecture. For instance, the service mostly doesn't use
value objects because few values are directly manipulated in the domain.

Concepts should be scaled up or down based on business complexity and expected
evolution of the application: if core is expected to only ever be accessed
through a web service, code from core handlers could be moved to HTTP handlers.
On the other hand, if core is to be exposed through multiple of tests, web,
gRPC, console, or a long-running service, the extra indirection with core
handlers becomes valuable.

The sample constraints itself to The Blue Book concepts. That means CQRS,
aggregates, entities, value objects, domain events, services, and so on. For the
HTTP API, the sample adheres to the Zalando API guidelines. It doesn't mean The
Blue Book and the Zalando API guidelines are the end all, be all, but the sample
reflects the constraints of a real-world service.

## Migrations

To create a new migration:

    $ make migrate-create

## Reflections

TBD.

## See also

- [Implementing Domain-Driven Design by Vaughn Vernon (The Blue Book)](https://www.amazon.com/Implementing-Domain-Driven-Design-Vaughn-Vernon/dp/0321834577).
- [Uncle Bob: Architecture the Lost Years](https://www.youtube.com/watch?v=WpkDN78P884).
- [.NET Microservices: Architecture for Containerized .NET Applications](https://docs.microsoft.com/en-us/dotnet/architecture/microservices), specifically the chapter on [Tackling Business Complexity in a Microservice with DDD and CQRS Patterns](https://docs.microsoft.com/en-us/dotnet/architecture/microservices/microservice-ddd-cqrs-patterns).
- [Zalando API guidelines](https://opensource.zalando.com/restful-api-guidelines).