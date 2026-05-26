package sql

import "testing"

func TestClassifySQL(t *testing.T) {
	cases := map[string]string{
		"SELECT * FROM users":           "SELECT",
		"  select id from users":        "SELECT",
		"INSERT INTO users VALUES (1)":  "INSERT",
		"\n\tupdate users SET x=1":      "UPDATE",
		"DELETE FROM users":             "DELETE",
		"WITH x AS (SELECT 1) SELECT *": "WITH",
		"":                              "",
		"   ":                           "",
		"BEGIN; SELECT 1; COMMIT;":      "BEGIN",
	}
	for q, want := range cases {
		if got := classifySQL(q); got != want {
			t.Errorf("classifySQL(%q) = %q, want %q", q, got, want)
		}
	}
}
