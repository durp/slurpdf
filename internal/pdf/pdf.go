package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ledongthuc/pdf"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

func Open(file string) (*os.File, *pdf.Reader, error) {
	return pdf.Open(file)
}

// GetPlainText returns all the text in the PDF file
func PlainText(r *pdf.Reader, pageRange PageRange) (reader io.Reader, err error) {
	pages := r.NumPage()
	if pageRange == AllPages {
		pageRange.Start = 1
		pageRange.End = pages
	}
	var buf bytes.Buffer
	fonts := make(map[string]*pdf.Font)
	for i := pageRange.Start; i <= pageRange.End; i++ {
		p := r.Page(i)
		for _, name := range p.Fonts() { // cache fonts so we don't continually parse charmap
			if _, ok := fonts[name]; !ok {
				logrus.Debugf("font: %s %s", name, p.Font(name).BaseFont())
				f := p.Font(name)
				fonts[name] = &f
			}
		}

		text, err := GetPlainText(p, fonts)
		//text, err := GetTextByColumns(p, fonts)
		if err != nil {
			return &bytes.Buffer{}, err
		}
		buf.WriteString(text)
	}
	return &buf, nil
}

var AllPages = PageRange{}

type PageRange struct {
	Start int
	End int
}

func GetTextByColumns(p pdf.Page, fonts map[string]*pdf.Font) (result string, err error) {
	var enc pdf.TextEncoding
	var nopEnc nopEncoder
	enc = &nopEnc

	var textBuilder bytes.Buffer
	showText := func(s string) {
		for _, ch := range enc.Decode(s) {
			_, err := textBuilder.WriteRune(ch)
			if err != nil {
				panic(err)
			}
		}
	}
	cols, err := p.GetTextByColumn()
	for _, col := range cols {
		for _, vert := range col.Content {
			showText(vert.S)
		}
	}
	return textBuilder.String(), nil
}

// GetPlainText returns all unformatted text of of a pdf
// fonts can be passed in (to improve parsing performance) or left nil
func GetPlainText(p pdf.Page, fonts map[string]*pdf.Font) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = ""
			err = errors.New(fmt.Sprint(r))
		}
	}()

	strm := p.V.Key("Contents")
	var enc pdf.TextEncoding
	var nopEnc nopEncoder
	enc = &nopEnc

	//if fonts == nil {
	//	fonts = make(map[string]*pdf.Font)
	//	for _, font := range p.Fonts() {
	//		f := p.Font(font)
	//		fonts[font] = &f
	//	}
	//}

	var textBuilder bytes.Buffer
	showText := func(s string) {
		for _, ch := range enc.Decode(s) {
			_, err := textBuilder.WriteRune(ch)
			if err != nil {
				panic(err)
			}
		}
	}

	pdf.Interpret(strm, func(stk *pdf.Stack, op string) {
		n := stk.Len()
		args := make([]pdf.Value, n)
		for i := n - 1; i >= 0; i-- {
			args[i] = stk.Pop()
		}

		switch op {
		default:
			// showText(fmt.Sprintf("%s: %v", op, args))
			return
		case "T*": // move to start of next line
			showText("\n")
		case "Tf": // set text font and size
			if len(args) != 2 {
				panic("bad Tf")
			}
			if font, ok := fonts[args[0].Name()]; ok {
				enc = font.Encoder()
			} else {
				enc = &nopEncoder{}
			}
		case "\"": // set spacing, move to next line, and show text
			if len(args) != 3 {
				logrus.Warnf("bad \\ operator")
			}
			showText(args[2].RawString())
		case "'":
			if len(args) != 1 {
				logrus.Warnf("bad ' operator")
			}
			showText(args[0].RawString())
		case "Tj":
			if len(args) != 1 {
				logrus.Warnf("bad Tj operator")
			}
			showText(args[0].RawString())
		case "TJ": // show text, allowing individual glyph positioning
			v := args[0]
			for i := 0; i < v.Len(); i++ {
				x := v.Index(i)
				if x.Kind() == pdf.String {
					showText(x.RawString())
				}
			}
		}

	})
	return textBuilder.String(), nil
}

type nopEncoder struct {
}

func (e *nopEncoder) Decode(raw string) (text string) {
	return raw
}
