/*
ratingsapp launches the main HTTP server that provides the ratingsapp API.

All configuration is passed as environment variables. The following ones are
available:
		RATINGSAPP_POSTGRES_DSL:
			mandatory, PostgreSQL connection string, e.g.
			postgres://user:pass@localhost:5432/ratingsappportal
		RATINGSAPP_JWT_SECRET:
			mandatory, JWT signing key to be used.
		PORT:
			mandatory, network port on which to serve the REST API.
*/
package main
