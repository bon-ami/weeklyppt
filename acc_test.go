package main

import (
	"fmt"
	"testing"

	"github.com/bon-ami/eztools"
)

func TestGetUsr(t *testing.T) {
	logger, db := prepareEnv()
	if logger != nil {
		defer logger.Close()
	}
	var (
		idE, idF   int
		errE, errF error
		output     string
	)

	//sub test env
	if db != nil {
		defer db.Close()
		idE, errE = getUsrFromEnv(db)
	}

	//sub test cfg file
	idF, output, errF = parseUsrFromFile(func(buf,
		id string) (output string, breaking bool) {
		return "matched", true
	}, func(buf, id string) (output string, breaking bool) {
		return
	}, 0)
	if errE != nil || errF != nil {
		fmt.Println(errE)
		fmt.Println(errF)
		t.Fail()
	}
	if db != nil {
		dispRes(t, "from env.", idE, output)
	}
	dispRes(t, "from cfg.", idF, output)
}

func dispRes(t *testing.T, from string, id int, output string) {

	switch id {
	case 0:
		t.Error(from + " NO ID")
	case eztools.InvalidID:
		t.Error(from + " Invalid ID")
	default:
		fmt.Println(from+" ID ", id, " ", output)
	}
}
