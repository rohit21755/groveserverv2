package graphql

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"

	"github.com/rohit21755/groveserverv2/graph"
	"github.com/rohit21755/groveserverv2/graph/generated"
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
)

// SetupGraphQLRoutes sets up GraphQL routes
func SetupGraphQLRoutes(r chi.Router, postgres *db.Postgres, redisClient *db.Redis, cfg *env.Config) {
	// Create resolver
	resolver := graph.NewResolver(postgres, redisClient, cfg)

	// Create GraphQL handler
	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(generated.Config{
			Resolvers: resolver,
		}),
	)

	// Configure transports
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	// WebSocket transport for subscriptions
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10,
	})

	// GraphQL endpoint
	r.Handle("/graphql", srv)

	// GraphQL Playground (for development)
	// Remove or protect this in production
	r.Handle("/playground", playground.Handler("GraphQL Playground", "/graphql"))
}
