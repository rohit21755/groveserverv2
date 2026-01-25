package graph

import (
	"github.com/rohit21755/groveserverv2/internal/db"
	"github.com/rohit21755/groveserverv2/internal/env"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	Postgres    *db.Postgres
	Redis       *db.Redis
	Config      *env.Config
}

// NewResolver creates a new resolver with dependencies
func NewResolver(postgres *db.Postgres, redis *db.Redis, cfg *env.Config) *Resolver {
	return &Resolver{
		Postgres: postgres,
		Redis:    redis,
		Config:   cfg,
	}
}
