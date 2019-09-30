package main

import "testing"

func TestGen(t *testing.T) {
	logger, db := prepareEnv()
	if logger != nil {
		defer logger.Close()
	}
	if db != nil {
		defer db.Close()
	}
	exportPpt(db)
}
