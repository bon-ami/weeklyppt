package main

import (
	"database/sql"
	"strconv"

	"github.com/bon-ami/eztools"
)

//parameters: member[0]:ID; [1]:name
func listTask1Member1Week(db *sql.DB, member []string, week int, weekDesc, table string, add bool) (skipMember bool) {
	for {
		selected, err := eztools.Search(db, eztools.TblWEEKLYTASKWORK,
			"contact="+member[0]+" AND week="+strconv.Itoa(week),
			[]string{eztools.FldID, eztools.FldSTR, "section"}, " ORDER BY section")
		if err != nil {
			eztools.LogErrPrint(err)
			return false
		}
		if len(selected) < 1 {
			eztools.ShowStrln("No current tasks for " + member[1] +
				", " + weekDesc + ".")
			break
		} else {
			eztools.ShowStrln("Current tasks for " + member[1] +
				", " + weekDesc + " listed as below:")
			for i, work := range selected {
				if len(work[1]) < 1 || len(work[2]) < 1 {
					eztools.LogPrint("zero length for " +
						member[0] + "'s task of week " +
						strconv.Itoa(week))
					continue
				}
				secID, err := eztools.GetPairStr(db, table, work[2])
				if err != nil {
					eztools.LogErrWtInfo("Processing member "+member[0]+"'s task in plan", err)
					continue
				}
				secDesc, err := eztools.GetPairStr(db, eztools.TblWEEKLYTASKTITLES, secID)
				if err != nil {
					eztools.LogErrWtInfo("Processing member "+member[0]+"'s task in titles", err)
					continue
				}
				des, err := eztools.GetPairStr(db, eztools.TblWEEKLYTASKDESC, work[1])
				if err != nil {
					eztools.LogErrWtInfo("Processing member "+member[0]+"'s description", err)
					continue
				}
				eztools.ShowStrln("\t" + strconv.Itoa(i) + ": " + secDesc + ". " + des)
			}
			if add {
				var skipWeek bool
				skipWeek, skipMember = queryTasks(db, table, member, selected)
				if skipMember {
					return
				}
				if skipWeek {
					break
				}
			}
		}
		if !add {
			return
		}
	}
	if !add {
		return
	}
	eztools.ShowStrln("Now, to add tasks.")
	for {
		horizontal(20)
		switch addTask(db, member, week, weekDesc, table) {
		case eztools.ErrInvalidInput:
			//skipMember = true
			return
		case nil:
		default:
			return
		}
	}
}

func member4weeks2Lst(db *sql.DB, member []string) {
	listTask1Member1Week(db, member, week, "this week", eztools.TblWEEKLYTASKCURR, false)
	listTask1Member1Week(db, member, week+1, "next week", eztools.TblWEEKLYTASKNEXT, false)
}

func chk(db *sql.DB, acc int) {
	members4weeks(db, acc, member4weeks2Lst)
}
