package main

import (
	"testing"
)

func TestFllSome(t *testing.T) {
	logger, db := prepareEnv()
	if logger != nil {
		defer logger.Close()
	}
	if db != nil {
		defer db.Close()
	}

	//ID 1 is the leader for tests
	fll(db, 1)
}
