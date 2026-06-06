// Package xlsx writes minimal, valid .xlsx (OOXML SpreadsheetML) files using only
// the standard library — no third-party dependencies. It supports a single sheet
// with string and numeric cells, which is all the telemetry export requires.
package xlsx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// Cell is one spreadsheet cell: either a string or a number.
type Cell struct {
	str   string
	num   float64
	isNum bool
}

// Str builds a text cell.
func Str(s string) Cell { return Cell{str: s} }

// Num builds a numeric cell.
func Num(f float64) Cell { return Cell{num: f, isNum: true} }

// Write serialises the given rows into an .xlsx document on w.
func Write(w io.Writer, sheetName string, rows [][]Cell) error {
	zw := zip.NewWriter(w)

	files := map[string]string{
		"[Content_Types].xml":        contentTypes,
		"_rels/.rels":                rootRels,
		"xl/workbook.xml":            workbookXML(sheetName),
		"xl/_rels/workbook.xml.rels": workbookRels,
		"xl/worksheets/sheet1.xml":   sheetXML(rows),
	}

	for name, body := range files {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(f, body); err != nil {
			return err
		}
	}
	return zw.Close()
}

const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`

const rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`

const workbookRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`

func workbookXML(sheetName string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="` + escape(sheetName) + `" sheetId="1" r:id="rId1"/></sheets>
</workbook>`
}

func sheetXML(rows [][]Cell) string {
	out := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`
	for r, row := range rows {
		out += fmt.Sprintf(`<row r="%d">`, r+1)
		for c, cell := range row {
			ref := colName(c) + strconv.Itoa(r+1)
			if cell.isNum {
				out += fmt.Sprintf(`<c r="%s"><v>%s</v></c>`, ref, strconv.FormatFloat(cell.num, 'f', -1, 64))
			} else {
				out += fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t xml:space="preserve">%s</t></is></c>`, ref, escape(cell.str))
			}
		}
		out += `</row>`
	}
	out += `</sheetData></worksheet>`
	return out
}

// colName converts a 0-based column index into A1 column letters (0→A, 26→AA).
func colName(idx int) string {
	name := ""
	for idx >= 0 {
		name = string(rune('A'+idx%26)) + name
		idx = idx/26 - 1
	}
	return name
}

func escape(s string) string {
	var b []byte
	buf := append(b, make([]byte, 0, len(s))...)
	w := &sliceWriter{&buf}
	_ = xml.EscapeText(w, []byte(s))
	return string(buf)
}

type sliceWriter struct{ b *[]byte }

func (w *sliceWriter) Write(p []byte) (int, error) {
	*w.b = append(*w.b, p...)
	return len(p), nil
}
