// auth-admin is an operator-only bootstrap tool. It never runs inside the API.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/QahNaaRlN/grace/internal/auth"
	"github.com/QahNaaRlN/grace/internal/platform/database"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "owner-invitation" {
		fmt.Fprintln(os.Stderr, "usage: auth-admin owner-invitation (DATABASE_URL required)")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot connect to database")
		os.Exit(1)
	}
	defer pool.Close()
	i, err := auth.NewRepository(pool).OwnerInvitation(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot create owner invitation (owner must be unlinked)")
		os.Exit(1)
	}
	// The invitation is a credential. Print once, deliberately, for the operator.
	if err = json.NewEncoder(os.Stdout).Encode(i); err != nil {
		os.Exit(1)
	}
}
