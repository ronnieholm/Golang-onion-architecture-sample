# TODO

- Split database access into writer and reader types and pass list of aggregates into writer to support changes to multiple aggregates in a single transaction.
- Have Makefile test call bin/tests/github.com-ronnieholm-resellerloyalty-test.test and pass in -rapid-checks=5 pass in CurrencyTests.
- Use slogger across infrastructure (see Golang presentation from a few years ago).
  - https://www.dash0.com/guides/golang-logging-libraries#1-slog
- Add Makefile target for code coverage report from running tests.
- Similar to codes on domain errors, validations could return a code like 1000 = name_too_long, inspired by Django (see Pydantic error messages)
- Use multierr for aggregating validation errors.
- Try GODEBUG=gctrace=1 ./myprogram.
- Use PostgreSQL as a Dead Letter Queue for Event-Driven Systems
  - https://www.diljitpr.net/blog-post-postgresql-dlq  
- Make use of 1.25 flight recorder feature: https://www.youtube.com/watch?v=mQM2DQ9yZ5I
- Enable Docker image with compiler to build application
  - docker run -v "$PWD":/app -w app go run main.go
  - Deployment should uses multi-stage Docker builds to minimise image size
- Setup GitHub CI pipeline
- For domain_events, add a correlation id column. That ID is set qhen the request comes in and should be included in every log entry.
- Add health check endpoint.
- Instead of cleaning up the database, implement the nested tx (savepoint) approach (from python video) so that tests can run in parallel
  without interfering with each other. How much faster is it?
  - Create an option to switch from nested tx to real transactions. Remember to flush savepoint to check most constraines
    that a normal commot would.
  - Create an option to preserve the crime scene (then stop on first test failure): 
      if failed:
        transaction.commit()  # Keep the evidence!
        print("\nTest failed: Data committed for debugging.")
     else:
        transaction.rollback() # Clean up as usual
- Move validation to middleware