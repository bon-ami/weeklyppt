package main

import (
	"database/sql"
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
	db       *sql.DB
	weekNum  int
	weekNext bool
	members  [][]string // [][0] ID, [][1] Name->week number, if reset
	buf      string
	bars     [barNum]string
	titles   *eztools.PairsStr
	section  int
}

func (p ppt2db) init(db *sql.DB, weeknum int, memberLst [][]string) {
	p.db = db
	p.weekNum = weeknum
	p.members = memberLst
	var err error
	for i := 0; i < barNum; i++ {
		p.bars[i], err = eztools.GetPairStrFromInt(db, eztools.TblWEEKLYTASKBARS, i)
		if err != nil {
			eztools.LogErrPrint(err)
			break
		}
	}
	p.titles, _ = eztools.GetSortedPairsStr(db, eztools.TblWEEKLYTASKTITLES)
}

// tran processes one string
func (p ppt2db) tran(s string) {
	switch {
	case strings.HasPrefix(s, p.bars[0]+" "):
		//this week
		p.weekNext = false
	case strings.HasPrefix(s, p.bars[1]+" "):
		//next week
		p.weekNext = true
		p.weekNum++
	//case strings.HasSuffix(s, ":"):
	//fallthrough
	default:
		//a title or task/name. get it in full and without colon
		if len(p.buf) > 0 {
			p.buf += s[:len(s)-1]
		} else {
			p.buf = s[:len(s)-1]
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
func (p ppt2db) tranTask(db *sql.DB, allAccounts, descIn string) {
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
	for account := range accounts {
		accNo, err := contacts.GetIDFromAllNames(db, account)
		if err != nil {
			eztools.LogErr(err)
			continue
		}
		// skip members
	}
}

// end marks end of a ppt2db
func (p ppt2db) end(db *sql.DB, endType int) {
	defer func() {
		p.buf = ""
	}()
	if week != p.weekNum && p.weekNext {
		// we process next week's plan for current week only
		return
	}
	switch endType {
	case endLine:
		// try to match a title
		if p.titles != nil && strings.HasSuffix(p.buf, ":") {
			i, err := p.titles.FindStr(p.buf)
			if err == nil {
				var table string
				switch p.weekNext {
				case true:
					table = eztools.TblWEEKLYTASKCURR
				case false:
					table = eztools.TblWEEKLYTASKNEXT
				}
				p.section, _ = eztools.GetPairIDFromInt(p.db, table, i)
				return
			}
		}
		// as a task
		delimInd := strings.IndexAny(p.buf, ":.")
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

func rd(db *sql.DB, file string, p2d *ppt2db) *presentation.Presentation {
	ppt, err := presentation.Open(file)
	if err != nil {
		eztools.LogErr(err)
		return nil
	}
	slides := ppt.Slides()
	if len(slides) < 1 {
		eztools.LogFatal("no slides")
	}
	defer p2d.end(db, endDoc)
	for r, k := range slides {
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
	return ppt
}

func importPpt(db *sql.DB, acc int) int {
	staticP, err := eztools.GetPairStr(db, eztools.TblCHORE, "WeeklyPptStaticPath")
	if err == nil {
		if acc == eztools.InvalidID {
			//acc is mandatory avoid override database of current group
			acc, err = chgUsr(db)
		}
		if acc == eztools.InvalidID {
			return acc
		}

		members, err := getMembers(db, acc)
		suf, err := eztools.GetPairStr(db, eztools.TblCHORE, "WeeklyPptStaticSuf")
		if err != nil {
			suf = ".pptx"
		}
		for i := week - 1; i <= week; i++ {
			var p2d ppt2db
			p2d.init(db, i, members)
			rd(db, staticP+strconv.Itoa(i)+suf, &p2d)
		}
	}
	return acc
}
