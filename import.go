package main

import (
	"database/sql"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/bon-ami/eztools"
	"github.com/bon-ami/eztools/contacts"
	"github.com/unidoc/unioffice/presentation"
)

const (
	barNum  = 2
	endLine = iota
	endPage
	endDoc
)

type ppt2db struct {
	db          *sql.DB
	weekNum     int
	weekNext    bool
	importNext  bool
	importAsked bool
	members     [][]string // [][0] ID, [][1] Name->week number, if reset
	buf         string
	bars        [barNum]string
	titles      *eztools.PairsStr
	section     int
}

func clrTskInWeek(db *sql.DB, weeknum int, memberLst [][]string) {
	if searched, err := eztools.Search(db, eztools.TblWEEKLYTASKWORK,
		"week="+strconv.Itoa(weeknum), nil, ""); err == nil && len(searched) == 0 {
		// no tasks in this week to clear
		return
	}
	var contacts string
	switch eztools.ChooseStrings([]string{"Do not delete for my members",
		"Delete for all members before syncing (default)",
		"Do not delete for any work from database"}) {
	case 0:
		for _, contact := range memberLst {
			if len(contacts) > 0 {
				contacts += " OR "
			}
			contacts += "contact!=\"" + contact[0] + "\""
		}
	case 2:
		return
	}
	if len(contacts) > 0 {
		contacts += " AND "
	}
	contacts += "week=" + strconv.Itoa(weeknum)
	if err := eztools.DeleteWtParams(db, eztools.TblWEEKLYTASKWORK, contacts); err != nil {
		eztools.LogErr(err)
	}
}

func (p *ppt2db) init(db *sql.DB, weeknum int, memberLst [][]string) {
	p.db = db
	p.weekNum = weeknum
	if eztools.PromptStr("Sync my team from file "+strconv.Itoa(weeknum)+
		" to database, as well as other team?(default=n/y)") != "y" {
		p.members = memberLst
	}
	var err error
	for i := 0; i < barNum; i++ {
		p.bars[i], err = eztools.GetPairStrFromInt(db, eztools.TblWEEKLYTASKBARS, i)
		if err != nil {
			eztools.LogErrPrint(err)
			break
		}
	}
	p.titles, _ = eztools.GetSortedPairsStr(db, eztools.TblWEEKLYTASKTITLES)
	clrTskInWeek(db, weeknum, memberLst)
}

// tran processes one string
func (p *ppt2db) tran(s string) {
	switch {
	case strings.HasPrefix(s, p.bars[0]+" "):
		//this week
		p.weekNext = false
		if eztools.Verbose > 1 {
			eztools.Log("this week detected")
		}
	case strings.HasPrefix(s, p.bars[1]+" "):
		//next week
		p.weekNext = true
		if eztools.Verbose > 1 {
			eztools.Log("next week detected")
		}
	//case strings.HasSuffix(s, ":"):
	//fallthrough
	default:
		//a title or task/name. get it in full and without colon
		//if eztools.Verbose > 1 {
		//eztools.Log("content before " + p.buf + " detected")
		//}
		if len(p.buf) > 0 {
			p.buf += s
		} else {
			p.buf = s
		}
		if eztools.Verbose > 1 {
			eztools.Log("content \"" + p.buf + "\" detected")
		}
	}
}

func trimSpaceFromSlice(str []string) (res []string) {
	for i, s := range str {
		trimmed := strings.Trim(s, " ")
		if len(trimmed) < 1 {
			return append(res, trimSpaceFromSlice(str[i+1:])...)
		}
		res = append(res, trimmed)
	}

	return
}
func (p *ppt2db) tranTask(db *sql.DB, allAccounts, descIn string) {
	if p.section == eztools.InvalidID {
		eztools.Log("NO valid section found for " + descIn)
		return
	}
	accounts := strings.Split(allAccounts, ",")
	desc := strings.Trim(descIn, " ")
	if len(desc) < 1 || len(accounts) < 1 {
		eztools.Log("accounts " + allAccounts + " or desc " + descIn + " EMPTY!")
		return
	}
	accounts = trimSpaceFromSlice(accounts)
	if len(accounts) < 1 {
		eztools.Log("accounts " + allAccounts + " EMPTY!")
		return
	}
	descID, err := addTaskDesc(p.db, desc)
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	weeknum := p.weekNum
	if p.weekNext {
		weeknum++
	}
ASS_ALL_ACC_4_TSK:
	for _, account := range accounts {
		accNo, err := contacts.GetIDFromAllNames(db, account)
		if err != nil {
			eztools.LogErr(err)
			continue
		}
		// skip members
		for _, member := range p.members {
			if member[0] == strconv.Itoa(accNo) {
				if eztools.Verbose > 1 {
					eztools.Log("skipping current group member " + member[1])
				}
				continue ASS_ALL_ACC_4_TSK
			}
		}
		if err = addTaskInt(db, descID, accNo, p.section, weeknum); err != nil {
			if err == eztools.ErrNoValidResults {
				eztools.Log("Task already exists in database: desc=" +
					strconv.Itoa(descID) + ", acc=" +
					strconv.Itoa(accNo) + ", sec=" +
					strconv.Itoa(p.section) + ", week=" +
					strconv.Itoa(weeknum))
			} else {
				eztools.LogErr(err)
			}
		}
	}
}

// end marks end of a ppt2db
func (p *ppt2db) end(db *sql.DB, endType int) {
	if len(p.buf) < 1 {
		return
	}
	defer func() {
		p.buf = ""
	}()
	if week != p.weekNum && !p.importNext && p.weekNext {
		if p.importAsked {
			return
		}
		p.importAsked = true
		// we process next week's plan for current week only
		if eztools.PromptStr("Sync week "+
			strconv.Itoa(p.weekNum+1)+
			", which was from old plan instead of week "+
			strconv.Itoa(p.weekNum)+"?(default=n/y)") != "y" {
			return
		}
		p.importNext = true
	}
	switch endType {
	case endLine:
		// try to match a title
		if p.titles != nil && strings.HasSuffix(p.buf, ":") {
			p.buf = p.buf[0 : len(p.buf)-1]
			i, err := p.titles.FindStr(p.buf)
			if err == nil {
				var table string
				switch p.weekNext {
				case true:
					table = eztools.TblWEEKLYTASKCURR
				case false:
					table = eztools.TblWEEKLYTASKNEXT
				}
				p.section, err = eztools.GetPairIDFromInt(p.db, table, i)
				if err != nil {
					eztools.LogErr(err)
					p.section = eztools.InvalidID
				}
				return
			}
		}
		// as a task
		delimInd := strings.IndexAny(p.buf, ":") // only separator we can descern
		if delimInd > 0 && delimInd < len(p.buf)-1 {
			// there is sth. on both sides of deliminator
			p.tranTask(db, p.buf[:delimInd], p.buf[delimInd+1:])
		} else {
			eztools.Log("Failed to parse: " + p.buf)
		}
	case endPage:
	case endDoc:
	default:
		eztools.Log("Ending a " + strconv.Itoa(endType) + "?")
	}
}

func rdByRD(db *sql.DB, rd io.ReaderAt, sz int64, p2d *ppt2db) {
	ppt, err := presentation.Read(rd, sz)
	if err != nil {
		eztools.LogErr(err)
		return
	}
	rdPPT(db, p2d, ppt)
}

func rdByFN(db *sql.DB, file string, p2d *ppt2db) {
	ppt, err := presentation.Open(file)
	if err != nil {
		eztools.LogErr(err)
		return
	}
	if eztools.Debugging && eztools.Verbose > 1 {
		eztools.Log("reading from " + file)
	}
	rdPPT(db, p2d, ppt)
}

func rdPPT(db *sql.DB, p2d *ppt2db, ppt *presentation.Presentation) {
	slides := ppt.Slides()
	if len(slides) < 1 {
		eztools.LogFatal("no slides")
	}
	defer p2d.end(db, endDoc)
	for r, k := range slides {
		p2d.section = eztools.InvalidID
		ph := k.PlaceHolders()
		if len(ph) < 1 {
			csld := k.X().CT_Slide.CSld
			if csld == nil {
				eztools.LogFatal("no csld")
			}
			if csld.SpTree == nil {
				eztools.LogFatal("no sptree")
			}
			for ist, st := range csld.SpTree.Choice {
				if st == nil {
					continue
				}
				//other choices not apprehendable
				for isp, sp := range st.Sp {
					if sp == nil {
						continue
					}
					if sp.TxBody != nil {
						for ip, p := range sp.TxBody.P {
							if p == nil {
								continue
							}
							for _, run := range p.EG_TextRun {
								if run == nil {
									continue
								}
								if run.R != nil {
									if eztools.Verbose > 2 {
										eztools.Log(strconv.Itoa(r) + "." +
											strconv.Itoa(ist) + "." +
											strconv.Itoa(isp) + "." +
											strconv.Itoa(ip) +
											" text run=" + run.R.T)
									}
									p2d.tran(run.R.T)
								}
								if run.Fld != nil {
									if run.Fld.T != nil {
										eztools.Log("text fld=" + *(run.Fld.T))
									}
								}
							}
							p2d.end(db, endLine)
						}
					}
				}
			}
		} else {
			for ii, i := range ph {
				pg := i.Paragraphs()
				if len(pg) < 1 {
					eztools.LogFatal("no pg")
				}
				eztools.ShowStrln("slide " + strconv.Itoa(r) + " pl " + strconv.Itoa(ii) + " #pg=" + strconv.Itoa(len(pg)))
				for pgi, j := range pg {
					tp := j.X()
					if tp == nil {
						eztools.LogFatal("no x")
					}
					tr := tp.EG_TextRun
					if tr == nil || len(tr) < 1 {
						eztools.ShowStrln("no tr")
						continue
					}
					for ti, t := range tr {
						trr := t.R
						trf := t.Fld
						if trr != nil {
							eztools.ShowStrln("textrun pg " + strconv.Itoa(pgi) + ",r " + strconv.Itoa(ti) + " = " + trr.T)
							trr.T = strings.Replace(trr.T, "Allen Tse", "monkey", 1)
						}
						if trf != nil && trf.T != nil {
							eztools.ShowStrln("textrun fld " + strconv.Itoa(ti) + " = " + *trf.T)
						}
					}
				}
			}
		}
		p2d.end(db, endPage)
	}
}

func importPpt(db *sql.DB, accI int, exclusive bool) (accO int, rd *os.File) {
	staticP, err := eztools.GetPairStr(db, eztools.TblCHORE, "WeeklyPptStaticPath")
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	if accI == eztools.InvalidID {
		//acc is mandatory avoid override database of current group
		accO, err = chgUsr(db)
		if err != nil {
			eztools.LogErrPrint(err)
		}
		if accO == eztools.InvalidID {
			return
		}
	} else {
		accO = accI
	}

	members, err := getMembers(db, accO)
	if err != nil {
		eztools.LogErrPrint(err)
	}
	suf, err := eztools.GetPairStr(db, eztools.TblCHORE, "WeeklyPptStaticSuf")
	if err != nil {
		suf = ".pptx"
	}
	for i := week - 1; i <= week; i++ {
		fn := staticP + strconv.Itoa(i) + suf
		fi, err := os.Stat(fn)
		if err != nil {
			if !os.IsNotExist(err) {
				eztools.LogErrWtInfo(fn, err)
			} else {
				eztools.Log(fn + " not exists")
				if exclusive && i == week {
					rd, err = os.OpenFile(fn, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
					if err != nil {
						eztools.LogPrint(fn + " not exists and not created")
					}
				}
			}
			continue
		}
		var p2d ppt2db
		p2d.init(db, i, members)
		if exclusive && i == week {
			rd, err = os.OpenFile(fn, os.O_RDWR, 0666)
			if err != nil {
				eztools.LogPrint(fn + " exists but not opened")
				continue
			}
			var wr *os.File
			wr, err = os.Create(fn + ".bak")
			if err != nil {
				eztools.LogPrint(fn + " back up file not created")
				continue
			}
			_, err = io.Copy(wr, rd)
			wr.Close()
			if err != nil {
				eztools.LogPrint(fn + " back up file created but not copied")
				continue
			}
			rdByRD(db, rd, fi.Size(), &p2d)
		} else {
			rdByFN(db, fn, &p2d)
		}
	}
	return
}
