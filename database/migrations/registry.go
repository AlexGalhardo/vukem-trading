// Package migrations holds the schema migrations.
package migrations

import (
	"github.com/AlexGalhardo/vukem/cache"
	"github.com/AlexGalhardo/vukem/schema"
	"github.com/AlexGalhardo/vukem/session"
)

// All lists the migrations in order; vukem make:migration appends here.
func All() []schema.Migration {
	return append(append(session.Migrations("sessions"), cache.Migrations("cache")...), []schema.Migration{
		m20260101000000CreatePostsTable(),
		m20261103130843CreateUsersTable(),
		m20261103130844CreateAuthTables(),
		m20261103130845CreateSocialAccountsTable(),
		terminalSchema(), artifactsSchema(), preferencesSchema(), recommendationsSchema(),
	}...)
}
