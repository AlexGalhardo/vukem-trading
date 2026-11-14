package reco

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/AlexGalhardo/vukem/database"
)

// Store persists daily reports; running twice on one date replaces that date.
type Store struct{ DB *database.DB }

// Save upserts the report under its date.
func (s Store) Save(ctx context.Context, rep Report) error {
	body, err := json.Marshal(rep)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, database.Rebind(s.DB.Dialect(),
		"INSERT INTO recommendations (id,payload,version,updated_at,status) VALUES (?,?,0,?,'ready') "+
			"ON CONFLICT (id) DO UPDATE SET payload=excluded.payload, version=recommendations.version+1, updated_at=excluded.updated_at"),
		rep.Date, string(body), time.Now().UTC().Format(time.RFC3339))
	return err
}

// Latest returns the newest report; ok is false when none exists yet.
func (s Store) Latest(ctx context.Context) (rep Report, ok bool, err error) {
	var body string
	err = s.DB.QueryRowContext(ctx, "SELECT payload FROM recommendations ORDER BY id DESC LIMIT 1").Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return Report{}, false, nil
	}
	if err != nil {
		return Report{}, false, err
	}
	return rep, true, json.Unmarshal([]byte(body), &rep)
}
