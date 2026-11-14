package migrations

import "github.com/AlexGalhardo/vukem/schema"

// recommendationsSchema stores one daily ranking report per UTC date. It serves the dashboard query
// "latest report" (ORDER BY id DESC LIMIT 1); the id is the YYYY-MM-DD date, so its primary key is the index.
// payload holds the JSON of reco.Report written by reco.Store.Save.
func recommendationsSchema() schema.Migration {
	return schema.Migration{ID: "20261114120000_create_recommendations", Up: func(s *schema.Schema) error {
		return s.Create("recommendations", func(t *schema.Table) {
			t.String("id").Primary()
			t.Text("payload")
			t.BigInteger("version").Default(0)
			t.String("updated_at")
			t.String("status").Default("ready")
		})
	}, Down: func(s *schema.Schema) error { return s.DropIfExists("recommendations") }}
}
