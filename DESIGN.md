# Design

## Core = Application + Domain

Instead of separate domain and application packages, they're combined into a
single core package. It avoids prefixing types and functions with "domain." and
duplicating file names, such as `domain/currency.go` and
`application/currency.go`. Often domain and application change together so
placing code in `core/currency.go` works out well.

Files become larger as an affect, but support more of a vertical slice
architecture.

## Value types

Instead of small value object, each with their own validation logic calling into
`validation.go`, validation happens at the core boundary. If in the domain value
objects were mutated a lot, the benefit would be repeated validation, but in
practice mutation in the domain is less common. Validation at the boundary means
less domain boilerplate, mapping in and out of domain types.

No value types is only a rule on thumb. Should a type with complex validation be
needed, one shouldn't refrain from creating specific domain types.

Each such value object would have a field called `V` rather than `Value` to not
pollute code with boilerplate too much. Each value object then needs a `New`
function to adhere to the same interface as today's validation functions.

## Naming conventions

- Events
  - Key fields should be spelled out, e.g., `CurrencyID` instead of `ID`.
  - Non-key fields shouldn't be prefixed by the entity name.
- Entities
  - Key fields should only be spelled out if they points to a foreign key.
- Command and queries
  - When containing more than one key field, spell out key field names.

## Third-party validation library in Core

Because Core is long-lived, one should avoid third-party dependencies. Only
Infrastructure should use third-party dependencies. Also, a validation library
becomes a hassle if/when it can only partially validates a struct and a custom
validate function is needed anyway.

Because Go doesn't have a nameof feature like in C#, compared to
reflection-based validation libraries, we manually must ensure field names in
error message stay in sync with field names in command and query structs.

## Integration tests

Avoid integration testing the validate function. Unless validation has control
flow logic, in almost all cases such tests would duplicate the validate function
itself. As each validate function is unit tested, it's reasonable to assume
validation works across a handler.

Instead integration tests should focus on the logic in the body of the handle
method.

As the database is considered an integral part of the application, it shouldn't
be mocked. Especially since bugs creep in when interacting with the database due
to malformed SQL or wrong serialization/deserialization.

Other dependencies such as a real time clock should generally be mocked as part
of the integration test setup.

## Unit of work, repositories, and change tracking

It's certainly possible to implement unit of work, repositories, and change
tracking patterns, but it's a lot of repetitive work for each aggregate. Go
doesn't have a full-fledged ORM supporting these features (ORM is considered
unidiomatic Go), so an alternative is needed.

One option is starting a transaction at the beginning of each request and
commit/rollback at the end of the request. The downside is that the underlying
connection is in use for the entirety of the request. It limits how many
requests we can have in flight and doesn't make effective use of connection
pooling.

It does mean, though, that we can use the database's transaction log as
intermediate state store. Any change to the database made during the request is
visible across any other query during the request. That's mostly useful for
publishing events across aggregates where there receiving handler queries for
up-to-date information.

Ignoring the latency cost of such requests/responses to the database server,
this approach alleviates the need for explicit change tracking, which in unit of
work with repositories is highly repetitive.

But note that for performance reasons a database usually defaults to transaction
isolation mode read committed". Committed changes from other transactions
become visible inside our ongoing transaction. Unless doing reporting queries,
that's usually not a problem (reporting should use transaction isolation mode
"snapshot" or "serializable").

By designing the application such that ongoing changes to one aggregate doesn't
have to become visible to another aggregate though the database, but explicitly
through function arguments, then instead of a single transaction across the
request, many smaller transactions may be used. Those smaller transactions would
make efficient use of the connection pool, similar to how an ORM typically
would. Finally changes to the database is collected as domain events and written
to the database in a single transaction.

A concurrency token for updates is required or update semantics would be "last
one wins". The concurrency token is a row version field included with the update
query. If version matches, records weren't changed by another transaction since
last read.

## References

- PGX Top to Bottom by Jack Christensen
  - https://www.youtube.com/watch?v=sXMSWhcHCf8
- Writing Clean and Efficient Table-Driven Unit Tests in Go
  - https://semaphore.io/blog/table-driven-unit-tests-go

