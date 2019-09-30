package main

import (
	"database/sql"
	"path"
	"strconv"

	"github.com/bon-ami/eztools"
	"github.com/bon-ami/eztools/contacts"

	"github.com/unidoc/unioffice/color"
	"github.com/unidoc/unioffice/drawing"
	"github.com/unidoc/unioffice/measurement"
	"github.com/unidoc/unioffice/presentation"
	"github.com/unidoc/unioffice/schema/soo/dml"
	"github.com/unidoc/unioffice/schema/soo/pml"
)

func addTB(slide *presentation.Slide, upper bool) *presentation.TextBox {
	tb := slide.AddTextBox()
	//tb.Properties().SetGeometry(dml.ST_ShapeTypeStar10)

	//ratio := 600 / 8.33
	tb.Properties().SetWidth(measurement.Inch * 13.33) //7.7 * measurement.Inch)
	if upper {
		tb.Properties().SetHeight(measurement.Inch * 0.59)
		tb.Properties().SetPosition(0, 0) //pos, pos)
		tb.Properties().SetSolidFill(color.RGB(47, 85, 151))
	} else {
		tb.Properties().SetHeight(measurement.Inch * 6.75)
		tb.Properties().SetPosition(0, measurement.Inch*0.59) //pos, pos)
	}
	//pos := measurement.Distance(i) * measurement.Inch
	return &tb
	//tb.Properties().LineProperties().SetSolidFill(color.White)
}

func addBanner(slide *presentation.Slide, title string) {
	tb := addTB(slide, true)
	p := tb.AddParagraph()
	p.Properties().SetAlign(dml.ST_TextAlignTypeCtr)

	r := p.AddRun()
	r.SetText(title)
	r.Properties().SetSize(32 * measurement.Point)
	r.Properties().SetFont("Calibri")
	r.Properties().SetBold(true)
	r.Properties().SetSolidFill(color.White)
}

func paragraphSpacing(p *drawing.Paragraph) {
	p.Properties().X().LnSpc = dml.NewCT_TextSpacing()
	p.Properties().X().LnSpc.SpcPct = dml.NewCT_TextSpacingPercent()
	var spcPer int32 = 120000
	p.Properties().X().LnSpc.SpcPct.ValAttr.ST_TextSpacingPercent = &spcPer
	p.Properties().X().SpcBef = dml.NewCT_TextSpacing()
	p.Properties().X().SpcBef.SpcPts = dml.NewCT_TextSpacingPoint()
	var spcPoi int32 = 600
	p.Properties().X().SpcBef.SpcPts.ValAttr = spcPoi
	var spcInd int32 = -120000 * 2
	p.Properties().X().IndentAttr = &spcInd
	hanging := true
	p.Properties().X().HangingPunctAttr = &hanging
	var spcHang int32 = 120000 * 2
	p.Properties().X().MarLAttr = &spcHang
}

func addParagraphTitle(tb *presentation.TextBox, size measurement.Distance, title string) {
	p := tb.AddParagraph()
	p.Properties().SetBulletChar("•")
	paragraphSpacing(&p)
	r := p.AddRun()
	r.SetText(title)
	r.Properties().SetSize(size * measurement.Point)
	r.Properties().SetFont("Calibri")
	r.Properties().SetBold(true)
}

func addParagraphCont(tb *presentation.TextBox, size measurement.Distance, cont string) {
	p := tb.AddParagraph()
	p.Properties().SetBulletChar("-")
	paragraphSpacing(&p)
	r := p.AddRun()
	r.SetText(cont)
	r.Properties().SetSize(size * measurement.Point)
	r.Properties().SetFont("Calibri")
}

func nameID2Str(db *sql.DB, id string) string {
	name, err := contacts.GetMail(db, id)
	if err != nil {
		eztools.LogErr(err)
		name = "?"
	}
	return name
}

func addCont(db *sql.DB, tb *presentation.TextBox, sectionID, sectionStr string, week1 int) {
	//p.Properties().SetAlign(dml.ST_TextAlignTypeCtr)
	addParagraphTitle(tb, 16, sectionStr+":")
	searched, err := eztools.Search(db, eztools.TblWEEKLYTASKWORK,
		"section="+sectionID+" AND week="+strconv.Itoa(week1),
		[]string{eztools.FldSTR, "contact", eztools.FldID},
		" ORDER BY "+eztools.FldSTR)
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	if len(searched) < 1 || len(searched[0]) < 1 {
		eztools.Log("No result found for " + sectionStr + ", week" + strconv.Itoa(week1))
		return
	}
	workID := searched[0][0]
	name := nameID2Str(db, searched[0][1])
	quan := len(searched)
	for i := 1; i < quan; i++ {
		if searched[i][0] == workID && i != 0 {
			name += ", " + nameID2Str(db, searched[i][1])
		}
		eztools.Log("user " + name + ": " + searched[i][0])
		if searched[i][0] != workID || i == quan-1 {
			add1Task(db, tb, name, workID)
			name = nameID2Str(db, searched[i][1])
			if searched[i][0] != workID && i == quan-1 {
				add1Task(db, tb, name, searched[i][0])
			}
			workID = searched[i][0]
		}
	}
}

func add1Task(db *sql.DB, tb *presentation.TextBox, name, workID string) {
	var (
		desc string
		err  error
	)
	if desc, err = eztools.GetPairStr(db, eztools.TblWEEKLYTASKDESC, workID); err != nil {
		eztools.LogErr(err)
		return
	}
	addParagraphCont(tb, 16, name+": "+desc)
}

func idInt2PgInd(id int) string {
	return strconv.Itoa(id/10 + 1)
}

func idStr2PgInd(id string) string {
	i, err := strconv.Atoi(id)
	if err != nil {
		return ""
	}
	return idInt2PgInd(i)
}

//parameters: sections[][0]: ID in table; [1]: str in table
func wrSlide(db *sql.DB, ppt *presentation.Presentation, table string, maxID int, sections [][]string) {
	slide := ppt.AddSlide()
	var (
		headline       string
		headnum, week1 int
	)
	switch table {
	case eztools.TblWEEKLYTASKCURR:
		headnum = 0
		week1 = week
	case eztools.TblWEEKLYTASKNEXT:
		headnum = 1
		week1 = week + 1
	}
	headline, err := eztools.GetPairStrFromInt(db, eztools.TblWEEKLYTASKBARS, headnum)
	if err == nil {
		eztools.LogErrPrint(err)
		return
	}
	addBanner(&slide, headline+" "+idStr2PgInd(sections[0][0])+"/"+idInt2PgInd(maxID))
	var tb *presentation.TextBox
	if len(sections) > 0 {
		tb = addTB(&slide, false)
	}
	for _, v := range sections {
		section, err := eztools.GetPairStr(db, eztools.TblWEEKLYTASKTITLES, v[1])
		if err != nil {
			eztools.LogErrPrint(err)
			continue
		}
		addCont(db, tb, v[0], section, week1)
	}
}

func wrWeek(db *sql.DB, ppt *presentation.Presentation, table string) {
	searched, err := eztools.Search(db, table, "", []string{eztools.FldID},
		" ORDER BY "+eztools.FldID+" DESC LIMIT 1")
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	if len(searched) < 1 {
		eztools.ShowStrln("No results found in " + table)
		return
	}
	maxID, err := strconv.Atoi(searched[0][0])
	if err != nil {
		eztools.ShowStrln("Max ID is not a number in " + table)
		return
	}
	if eztools.Debugging {
		eztools.Log("Max ID=" + strconv.Itoa(maxID))
	}
	for i := 10; i < (maxID + 10); i += 10 {
		searched, err = eztools.Search(db, table,
			eztools.FldID+">"+strconv.Itoa(i-10)+" AND "+
				eztools.FldID+"<"+strconv.Itoa(i+1),
			[]string{eztools.FldID, eztools.FldSTR},
			" ORDER BY "+eztools.FldID)
		if err != nil {
			eztools.LogErrPrint(err)
			break
		}
		if len(searched) < 1 {
			continue
		}
		wrSlide(db, ppt, table, maxID, searched)
	}
}

func wr(db *sql.DB, file string) {
	ppt := presentation.New()
	ppt.X().CT_Presentation.SldSz = pml.NewCT_SlideSize()
	ppx := ppt.X().CT_Presentation.SldSz
	ppx.TypeAttr = pml.ST_SlideSizeTypeScreen16x9
	ppx.CxAttr = 12192000
	ppx.CyAttr = 6858000
	wrWeek(db, ppt, eztools.TblWEEKLYTASKCURR)
	wrWeek(db, ppt, eztools.TblWEEKLYTASKNEXT)
	if err := ppt.Validate(); err != nil {
		eztools.LogErrFatal(err)
	}
	if err := ppt.SaveToFile(file + ".pptx"); err != nil {
		eztools.LogErrFatal(err)
	}
}

func exportPpt(db *sql.DB) {
	if staticP, err := eztools.GetPairStr(db, eztools.TblCHORE,
		"WeeklyPptStaticPath"); err == nil {
		wr(db, path.Base(staticP)+strconv.Itoa(week))
	}
}
