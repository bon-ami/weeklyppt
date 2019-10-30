package main

import (
	"database/sql"
	"testing"
)

func TestMng(t *testing.T) {
	mainLoop(func(*sql.DB, int) (su, mng bool) {
		return false, true
	})
}
