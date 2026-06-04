package sqlclass

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantVerb  string
		wantTable string
	}{
		{"select-from", "SELECT * FROM users WHERE id = $1", "SELECT", "users"},
		{"select-schema-qualified", `SELECT * FROM "public"."users"`, "SELECT", "users"},
		{"insert", "INSERT INTO orders (id, total) VALUES ($1, $2)", "INSERT", "orders"},
		{"update", "UPDATE accounts SET balance = $1 WHERE id = $2", "UPDATE", "accounts"},
		{"delete", "DELETE FROM sessions WHERE id = $1", "DELETE", "sessions"},
		{"merge", "MERGE INTO targets t USING …", "MERGE", "targets"},
		{"with-cte", "WITH t AS (SELECT 1) SELECT * FROM t", "WITH", ""},
		{"leading-line-comment", "-- ignore\nSELECT 1 FROM dual", "SELECT", "dual"},
		{"leading-block-comment", "/* hi */ INSERT INTO logs (msg) VALUES ($1)", "INSERT", "logs"},
		{"backticked-table", "SELECT * FROM `events`", "SELECT", "events"},
		{"bracketed-table", "SELECT * FROM [events]", "SELECT", "events"},
		{"lowercased-keyword", "select * from items", "SELECT", "items"},
		{"empty", "", "", ""},
		{"garbage", "this is not sql", "THIS", ""},
		{"select-no-from", "SELECT 1", "SELECT", ""},
		{"column-contains-keyword", "SELECT fromage FROM cheeses", "SELECT", "cheeses"},
		{"keyword-only-as-suffix", "SELECT prefrom, x FROM t", "SELECT", "t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.in)
			if got.Verb != tc.wantVerb {
				t.Errorf("Verb = %q, want %q", got.Verb, tc.wantVerb)
			}
			if got.Table != tc.wantTable {
				t.Errorf("Table = %q, want %q", got.Table, tc.wantTable)
			}
		})
	}
}
