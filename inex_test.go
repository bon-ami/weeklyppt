package main

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

func TestInEx(t *testing.T) {
	logger, db := prepareEnv()
	if logger != nil {
		defer logger.Close()
	}

	const (
		fn = "\\\\10.75.10.81\\Share\\Team\\FW&APP\\Documents\\test.ppt"
		sl = 30 * time.Second
	)
	var p2d ppt2db
	p2d.init(db, week, nil)
	fmt.Println("")
	fp, err := os.OpenFile(fn, os.O_RDWR, 0666)
	if err != nil {
		fmt.Println(fn + " NOT FOUND!")
		t.FailNow()
	}
	fi, err := os.Stat(fn)
	if err != nil {
		fmt.Println(fn + " NOT readable!")
		t.FailNow()
	}
	if fp != nil {
		rdByRD(db, fp, fi.Size(), &p2d)
		fmt.Println("")
		fmt.Println(fn + " locked for " + sl.String())
		time.Sleep(sl)
		_, err = fp.Seek(0, io.SeekStart)
		if err == nil {
			err = fp.Truncate(0)
			if err == nil {
				wrByWR(db, fp)
			}
		}
		fp.Close()
		if err != nil {
			t.Fail()
		}
	}
}
