package main

import (
	"database/sql"
	"strconv"

	"github.com/bon-ami/eztools"
)

func modTaskExec(db *sql.DB, id string, str int) {
	if err := eztools.UpdatePairWtParams(db,
		eztools.TblWEEKLYTASKWORK, id, strconv.Itoa(str)); err != nil {
		eztools.LogErr(err)
	}
}

//parameters: member[0]:ID; [1]:name;	id[0]: ID; [1]: str; [2]: section
func modTask(db *sql.DB, table string, member, id []string) {
	secStr := id[2]
	section, err := chooseSection(db, table,
		"Choose a section if you want to move this task. Invalid to keep current")
	if err == nil {
		prevSec, err := strconv.Atoi(id[2])
		if err == nil && prevSec != section {
			err = eztools.UpdateWtParams(db, eztools.TblWEEKLYTASKWORK,
				eztools.FldID+"="+id[0], []string{"section"},
				[]string{strconv.Itoa(section)}, false)
			if err != nil {
				eztools.LogErrPrint(err)
			} else {
				eztools.ShowStrln("moved.")
			}
			secStr = strconv.Itoa(section)
		}
	}
	descID, err := chooseOrAddTaskDesc(db, secStr)
	if err == nil {
		modTaskExec(db, secStr, descID)
	}
}

func chooseOrAddTaskDesc(db *sql.DB, section string) (descID int, err error) {
	works, err := eztools.Search(db, eztools.TblWEEKLYTASKDESC, "",
		[]string{"DISTINCT d.id", "d.str"},
		" d JOIN "+eztools.TblWEEKLYTASKWORK+
			" w ON w.section="+section+
			" AND w.str=d.id")
	if err != nil {
		eztools.LogErrPrint(err)
	} else {
		descID = eztools.ChooseInts(works,
			"Choose from tasks in existence (invalid to create one)")
		if descID != eztools.InvalidID {
			return
		}
	}
	descID, err = promptTaskDesc(db)
	return
}

func addTaskDesc(db *sql.DB, desc string) (descID int, err error) {
	descID, err = eztools.Locate(db, eztools.TblWEEKLYTASKDESC, desc)
	if err != nil || descID != eztools.DefID {
		//error or one in existence
		return
	}
	descID, err = eztools.AddPairNoID(db, eztools.TblWEEKLYTASKDESC, desc)
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	return
}

func promptTaskDesc(db *sql.DB) (descID int, err error) {
	desc := eztools.PromptStr("Input description for this task (invalid value to return)")
	if len(desc) < 1 {
		err = eztools.ErrInvalidInput
		return
	}
	return addTaskDesc(db, desc)
}

func chooseSection(db *sql.DB, table, desc string) (section int, err error) {
	sections, err := eztools.Search(db,
		eztools.TblWEEKLYTASKTITLES+","+table,
		eztools.TblWEEKLYTASKTITLES+".id="+table+".str",
		[]string{"DISTINCT " + eztools.TblWEEKLYTASKTITLES + ".id",
			eztools.TblWEEKLYTASKTITLES + ".str"}, "")
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	section = eztools.ChooseInts(sections, desc)
	if section == eztools.InvalidID {
		err = eztools.ErrInvalidInput
		return
	}
	section, err = eztools.GetPairID(db, table, strconv.Itoa(section))
	return
}

//parameters: member[0]:ID; [1]:name
func addTask(db *sql.DB, member []string, week int, weekDesc, table string) (err error) {
	section, err := chooseSection(db, table,
		"Choose one section for the new task for "+
			member[1]+" of "+weekDesc)
	if err != nil {
		return
	}
	secStr := strconv.Itoa(section)
	desc, err := chooseOrAddTaskDesc(db, secStr)
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	_, err = eztools.AddWtParamsUniq(db, eztools.TblWEEKLYTASKWORK,
		[]string{eztools.FldSTR, "contact", "section", "week"},
		[]string{strconv.Itoa(desc), member[0], secStr, strconv.Itoa(week)}, true)
	if err != nil {
		eztools.LogErrPrint(err)
	}
	return
}

func promptID(notif string, len int) (id int, err error) {
	id, err = eztools.PromptInt("Which task to " + notif + "?")
	if err != nil || id < 0 || id >= len {
		return id, eztools.ErrInvalidInput
	}
	return
}

//parameters: member[0]:ID; [1]:name;	id[][0]: ID; [1]: str; [2]: section
func queryTasks(db *sql.DB, table string, member []string, id [][]string) (skipWeek, skipMember bool) {
	choices := []string{
		"Change a task above",                  //0
		"Delete a task above",                  //1
		"Skip modifying tasks above (default)", //2
		"Skip this member"}                     //3
	var (
		err error
		i   int
	)
	switch eztools.ChooseStrings(choices) {
	case 0:
		i, err = promptID("modify", len(id))
		if err == nil {
			modTask(db, table, member, id[i])
		}
	case 1:
		i, err = promptID("delete", len(id))
		if err == nil {
			err = eztools.Delete(db, eztools.TblWEEKLYTASKWORK, id[i][0])
		}
	case 3:
		skipMember = true
	default:
		skipWeek = true
	}
	if err != nil {
		eztools.LogErrPrint(err)
	}
	return
}

func member4weeks2Add(db *sql.DB, member []string) {
	//if week < 2 {
	//eztools.ShowStrln("No reference plan for the first week of a year.")
	//}
	if listTask1Member1Week(db, member, week, "this week", eztools.TblWEEKLYTASKCURR, true) {
		return
	}
	horizontal(10)
	listTask1Member1Week(db, member, week+1, "next week", eztools.TblWEEKLYTASKNEXT, true)
}

func fll(db *sql.DB, acc int) {
	members4weeks(db, acc, member4weeks2Add)
}
