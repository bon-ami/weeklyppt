package main

import (
	"database/sql"
	"strconv"

	"github.com/bon-ami/eztools"
)

func listTitles1(db *sql.DB, tbl string) {
	pi, err := eztools.GetSortedPairsInt(db, tbl)
	if err != nil {
		eztools.LogErr(err)
		return
	}
	var (
		id, val int
		desc    string
	)
	for {
		if id, val, err = pi.Next(); err != nil {
			break
		}
		desc, err = eztools.GetPairStrFromInt(db, eztools.TblWEEKLYTASKTITLES, val)
		if err != nil {
			eztools.LogErr(err)
			continue
		}
		eztools.ShowStrln("ID " + strconv.Itoa(id) + ". Page " + strconv.Itoa(id/10) + ": " + desc)
	}
}

func chkTitle41Page(db *sql.DB, tbl, desc string) (int, error) {
	idC, err := eztools.PromptInt("ID for " + desc + " page")
	if err != nil {
		eztools.LogErr(err)
		return eztools.InvalidID, err
	}
	id, err := eztools.GetPairStrFromInt(db, tbl, idC)
	if err != nil {
		if err == eztools.ErrNoValidResults {
			err = nil
		} else {
			eztools.LogErr(err)
		}
	} else {
		eztools.ShowStrln("ID " + id + " already exists!")
		err = eztools.ErrInvalidInput
	}
	return idC, err
}

func addTitle(db *sql.DB) {
	var (
		idS, idI int
		err      error
	)
	str := eztools.PromptStr("Title name")
	if len(str) < 1 {
		return
	}
	idS, err = eztools.AddPairNoID(db, eztools.TblWEEKLYTASKTITLES, str)
	if err != nil {
		return
	}
	idI = eztools.InvalidID
	defer func() {
		if err != nil {
			//TODO: lock the table? Tx does not work, since idS needed for these two additions
			if idS != eztools.InvalidID {
				if err = eztools.DeleteWtID(db, eztools.TblWEEKLYTASKTITLES, strconv.Itoa(idS)); err != nil {
					eztools.LogErr(err)
				}
			}
			if idI != eztools.InvalidID {
				if err = eztools.DeleteWtID(db, eztools.TblWEEKLYTASKCURR, strconv.Itoa(idI)); err != nil {
					eztools.LogErr(err)
				}
			}
		}
	}()
	if idI, err = addTitle2Layout(db, eztools.TblWEEKLYTASKCURR, "current", idS); err != nil {
		return
	}
	_, err = addTitle2Layout(db, eztools.TblWEEKLYTASKNEXT, "next", idS)
}

func addTitle2Layout(db *sql.DB, table, desc string, title int) (idI int, err error) {
	idC, err := chkTitle41Page(db, table, desc)
	if err != nil {
		return
	}
	idI, err = eztools.AddPair(db, table, idC, strconv.Itoa(title))
	return
}

func chgTitle(db *sql.DB) {
	id, err := eztools.ChoosePair(db, eztools.TblWEEKLYTASKTITLES)
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	str := eztools.PromptStr("Change it to?")
	if len(str) > 0 {
		err = eztools.UpdatePairWtParams(db,
			eztools.TblWEEKLYTASKTITLES, strconv.Itoa(id), str)
		if err != nil {
			eztools.LogErrPrint(err)
		}
	}
}

func chgLayout1Add(db *sql.DB, table string, title int) {
	if title == eztools.InvalidID {
		addTitle(db)
	} else {
		if _, err := addTitle2Layout(db, table, "", title); err != nil {
			eztools.LogErr(err)
		}
	}
}

func chgLayout1Mod(db *sql.DB, table string, id, newID int) {
	_, err := eztools.GetPairStrFromInt(db, table, newID)
	if err == nil {
		eztools.ShowStrln("ID already exists!")
		return
	}
	if err != eztools.ErrNoValidResults {
		eztools.LogErrPrint(err)
		return
	}
	idSOld := strconv.Itoa(id)
	idSNew := strconv.Itoa(newID)
	if err = eztools.UpdatePairID(db, table,
		idSOld, idSNew); err != nil {
		eztools.LogErrPrint(err)
		return
	}
	if err = eztools.UpdateWtParams(db, table, eztools.FldSECTION+"="+idSOld,
		[]string{eztools.FldSECTION},
		[]string{idSNew}, true); err != nil {
		eztools.LogErrPrint(err)
		return
	}
}

//operations are same on both curr and next tables
func chgLayout1(db *sql.DB, table string) {
	eztools.ShowStrln("Which title to change? (IDs are before colons)")
	id, err := eztools.ChoosePairOrAdd(db, table, true)
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	if id == eztools.InvalidID {
		//add an item
		eztools.ShowStrln("Choose a title in existence or add one.")
		title, err := eztools.ChoosePairOrAdd(db, eztools.TblWEEKLYTASKTITLES, true)
		if err != nil {
			eztools.LogErrPrint(err)
			return
		}
		chgLayout1Add(db, eztools.TblWEEKLYTASKCURR, title)
		chgLayout1Add(db, eztools.TblWEEKLYTASKNEXT, title)
	} else {
		//move an item
		newID, err := eztools.PromptInt("new ID=")
		if err != nil {
			return
		}
		chgLayout1Mod(db, eztools.TblWEEKLYTASKCURR, id, newID)
		chgLayout1Mod(db, eztools.TblWEEKLYTASKNEXT, id, newID)
	}
}

func chgLayout(db *sql.DB) {
	//switch eztools.PromptStr("Current page or Next?") {
	//case "c", "C":
	chgLayout1(db, eztools.TblWEEKLYTASKCURR)
	//case "n", "N":
	//chgLayout1(db, eztools.TblWEEKLYTASKNEXT)
	//}
}

func mnt(db *sql.DB) {
	eztools.ShowStrln("Current week's pages,")
	listTitles1(db, eztools.TblWEEKLYTASKCURR)
	eztools.ShowStrln("Next week's pages,")
	listTitles1(db, eztools.TblWEEKLYTASKNEXT)
	choices := []string{"Return", //0
		"Add a title",    //1
		"Modify a title", //2
		"Delete a title", //3
		"Modify layout"}  //4
	c := eztools.ChooseStrings(choices)
	switch c {
	case 0:
	case 1:
		addTitle(db)
	case 2:
		chgTitle(db)
	case 4:
		chgLayout(db)
	default:
		eztools.ShowStrln("TODO")
	}
}
