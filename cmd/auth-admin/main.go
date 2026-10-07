// auth-admin is an operator-only bootstrap tool. It never runs inside the API.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/QahNaaRlN/grace/internal/auth"
	"github.com/QahNaaRlN/grace/internal/platform/access"
	"github.com/QahNaaRlN/grace/internal/platform/database"
)

func main() {
	migration := len(os.Args) == 3 && os.Args[1] == "prepare-owner-email"
	if !migration && (len(os.Args) != 2 || os.Args[1] != "owner-invitation") {
		fmt.Fprintln(os.Stderr, "usage: auth-admin owner-invitation | prepare-owner-email EMAIL (DATABASE_URL required)")
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
	if migration {
		i, e := auth.NewRepository(pool).EmailMigrationInvitation(ctx, access.BootstrapOwnerID, os.Args[2])
		if e != nil {
			fmt.Fprintln(os.Stderr, "cannot prepare owner email migration")
			os.Exit(1)
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"prepared": true, "expiresAt": i.Expires})
		return
	}
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
