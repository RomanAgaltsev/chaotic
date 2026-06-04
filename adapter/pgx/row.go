package pgx

// chaosRow satisfies pgxv5.Row and returns the carried error on Scan.
// Used when a chaos rule fires on QueryRow — pgx's contract is that
// QueryRow always returns a non-nil Row whose error surfaces from Scan.
type chaosRow struct {
	err error
}

func (r chaosRow) Scan(dest ...any) error {
	return r.err
}
