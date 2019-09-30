package main

import (
	"database/sql"
	"flag"
	"os"
	"path"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/bon-ami/eztools"
	"github.com/bon-ami/eztools/contacts"
)

var (
	ver  string
	week int
)

// return values: [][0] ID, [][1] Name
func getMembers(db *sql.DB, acc int) ([][]string, error) {
	if acc == eztools.InvalidID {
		acc, err := chgUsr(db)
		if (err != nil && err == errNotSaved) || acc == eztools.InvalidID {
			return nil, err
		}
	}
	return contacts.GetMembers(db, acc)

}

type member4weeksFn func(*sql.DB, []string)

func members4weeks(db *sql.DB, acc int, fc member4weeksFn) {
	members, err := getMembers(db, acc)
	if err != nil || members == nil {
		return
	}
	for _, member := range members {
		horizontal(30)
		fc(db, member)
	}
}

func horizontal(level int) {
	if level < 1 {
		level = 30
	}
	for i := 0; i < level; i++ {
		eztools.ShowStr("-")
	}
	eztools.ShowStrln("")
}

func chkManager(db *sql.DB, acc int) bool {
	switch acc {
	case 1: //super user
		return true
	case eztools.InvalidID:
		return false
	}
	searched, err := eztools.Search(db, eztools.TblTEAM,
		eztools.FldLEADER+"="+strconv.Itoa(acc), nil, "")
	if err != nil {
		eztools.LogErr(err)
		return false
	}
	if len(searched) > 0 {
		return true
	}
	return false
}

func prepareEnv() (logger *os.File, db *sql.DB) {
	_, week = time.Now().ISOWeek()
	eztools.ShowStrln("V" + ver + ". Now it is week " + strconv.Itoa(week))
	home, _ := os.UserHomeDir()
	logger, err := os.OpenFile(path.Join(home, "WeeklyPpt.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err == nil {
		if err = eztools.InitLogger(logger); err != nil {
			eztools.ShowStrln(err.Error())
			logger.Close()
			logger = nil
		}
	} else {
		eztools.ShowStrln("Failed to open log file")
	}

	if len(ver) < 1 {
		ver = "dev"
	}
	if ver == "dev" {
		eztools.Debugging = true
		flagV := flag.Int("v", 1, "verbose level")
		flag.Parse()
		eztools.Verbose = *flagV
		eztools.ShowStrln("verbose " + strconv.Itoa(eztools.Verbose))
	} else {
		flagH := flag.Bool("h", false, "help messages")
		flag.Parse()
		if *flagH {
			eztools.ShowStrln("V1.1 account detection, verification and encryption on saving")
			eztools.ShowStrln("V1.0 initial release")
			return
		}
	}
	db, err = eztools.Connect()
	if err != nil {
		eztools.LogErrFatal(err)
	}
	return
}

func main() {
	logger, db := prepareEnv()
	if logger != nil {
		defer logger.Close()
	}
	if db != nil {
		defer db.Close()
	} else {
		return
	}

	upch := make(chan bool)
	svch := make(chan string)
	go eztools.AppUpgrade(db, "WeeklyPpt", ver, &svch, upch)

	choices := []string{"quit", //0
		"Fill/Edit this week's report (default)", //1
		"List this week's report",                //2
		"Generate report"}                        //3
	eztools.ShowStrln("checking for server and syncing between servers...")
	serverGot := <-upch
	if serverGot {
		<-svch
	}

	acc, fixed, err := getUsr(db)
	eztools.ShowStrln("")
	if err != nil {
		acc = eztools.InvalidID
		fixed = false
	}
	if !fixed {
		choices = append(choices, "Change account")
	}

	acc = importPpt(db, acc)
	chkManagerNeeded := true
	chkManagerAdded := false
EXIT:
	for {
		if chkManagerNeeded {
			if !chkManagerAdded {
				if chkManager(db, acc) {
					choices = append(choices, "Maintain report structure") //5
				}
			} else {
				if !chkManager(db, acc) {
					choices = choices[:len(choices)-1]
				}
			}
		}
		chkManagerNeeded = false
		c := eztools.ChooseStrings(choices)
		switch c {
		case 1, eztools.InvalidID, 2:
			if acc != eztools.InvalidID {
				break
			}
			//acc is mandatory
			acc, err = chgUsr(db)
			if err == nil || err == errNotSaved {
				chkManagerNeeded = true
			} else {
				c = 0
			}
		}
		switch c {
		case 0:
			break EXIT
		case 1, eztools.InvalidID:
			fll(db, acc)
		case 2:
			chk(db, acc)
		case 3:
			exportPpt(db)
		case 4:
			if !fixed {
				acc, err = chgUsr(db)
				if err == nil {
					chkManagerNeeded = true
				} else {
					eztools.LogErrPrint(err)
				}
			} else {
				mnt(db)
			}
		case 5:
			if !fixed {
				mnt(db)
			}
		default:
			eztools.LogPrint("impossible choice: " + strconv.Itoa(c))
		}
		horizontal(0)
	}

	if serverGot {
		eztools.ShowStrln("waiting for update check to end...")
		<-upch
	}
}
