Overview
========

RatingsApp is designed to allow authentication and authorization of users to a rating system. As well as the ability to CRUD a database of ratings.

This system is able to detect which user is interacting with the system and allow you to make any transaction in the database as long as the user has the relevant privileges.

Once you have a considerable number of ratings, you can list one by one or by target, which points to a product or object to which the rating is directed.


Documentation
=============

- [Authentication and Authorization](Authentication.md) 🔑
- [Rating](Rating.md) ★


Development
===========

To run the environment in Docker:

    docker-compose run --rm --service-ports --name ratingsapp ratingsapp

To build and run the backend, run this on the Go container shell:

    go build -mod=vendor ./cmd/ratingsapp && ./ratingsapp -v

To run your build locally, if outside your GOPATH:

    go build ./cmd/ratingsapp && RATINGSAPP_POSTGRES_DSL=postgres://ratingsapp:ratingspassword@localhost:5432/ratingsapp?sslmode=disable ./ratingsapp -v

When running integration tests, the RATINGSAPP_POSTGRES_DSL_TEST must be defined pointing to a testing Postgres instance. This is set already on .drone.yml and on docker-compose.

Don't forget to stop the database container after you're done:

    docker-compose stop

TIP: You can reset the whole docker-compose service with the following command. Use it carefully.

    docker-compose down && docker-compose up -d


Tests
-----

In the docker-compose container shell, you may execute the tests by issuing:

    go test -mod=vendor -cover -v -p 1 -count=1 -coverprofile=cover.out ./internal/...

The -count=1 makes sure the test cache is not used. The packages in internal and the e2etests must be tests separately; this prevents issues where multiple packages make use of the test database.

You can check a web-based coverage report by launching the following command on the **host**:

    go tool cover -html=cover.out

Debug backend
-------------

Configure your VSCode to debug Go code using Delve (check on VSCode wiki). Make sure you set the right environment variables with the database connection details. It's easier to debug the backend locally, so you'll need all Go 1.12 tooling installed.


Migrations
----------

This project uses GORM Migration tool and therefore Auto Migration is active. This will automatically migrate your schema, to keep your schema update to date

WARNING: AutoMigrate will ONLY create tables, missing columns and missing indexes, and WON’T change existing column’s type or delete unused columns to protect your data.


Vendoring
---------

This project uses the Go 1.11 module functionality. If running it from inside your GOPATH, please set the environment variable GO111MODULE=on to manage the dependencies correctly. When adding new dependencies, please vendor them with `go mod vendor`. Please also commit the vendored repositories that are added.


Deployment for testing
==========

    docker-compose run --rm --service-ports --name ratingsapp ratingsapp

This will result OK if everything works and we will be inside of the docker container.

...
Two options from here:

1. Build the project to test the API from a client like *Postman*.

    `go build -mod=vendor ./cmd/ratingsapp && ./ratingsapp -v`

2. With go test we can test the components separately.

    `go test -mod=vendor -cover -v -p 1 -count=1 ./...`


Hot reload
----------

To watch the Go files for changes and automatically recompile and reload the server, use the ["realize"](https://github.com/oxequa/realize) utility:

    go get github.com/oxequa/realize

    $ realize start

The utility will not die if it fails to build.

TIP: Remember to clean (drop and create) the database if a significant change in the schema (like a unique constraint) is made. Realize will not do it for you.


Deployment
==========

Environment variables
---------------------

- **RATINGSAPP_POSTGRES_DSL**: PostgreSQL connection string, e.g. `postgresql://ratingsapp:ratingspassword@postgres:5432/admin`
- **RATINGSAPP_JWT_SECRET**: The JWT signing key to be used. A default development value will be used if not defined.
- **PORT**: TCP port the HTTP server will listen to. Defaults to `8000`.
