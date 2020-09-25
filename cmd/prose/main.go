package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"slurpdf/internal/pdf"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/jdkato/prose/v3"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app  = kingpin.New("prose", "consume a pdf file and create a summary of the digested tokens")
	args = struct {
		input     *string
		tags      *bool
		entities  *bool
		sentences *bool
		fileType  *string
	}{
		input:     app.Flag("in", "input file to process").Short('i').Required().String(),
		fileType:  app.Flag("type", "inout file type (pdf, txt)").Short('x').Required().String(),
		tags:      app.Flag("tags", "display tag details").Short('t').Bool(),
		entities:  app.Flag("entities", "display entity details").Short('e').Bool(),
		sentences: app.Flag("sentences", "display sentence details").Short('s').Bool(),
	}
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	kingpin.MustParse(app.Parse(os.Args[1:]))
	logrus.Infof("input: %s", *args.input)

	var r io.Reader
	if *args.fileType == "pdf" {
		f, slurper, err := pdf.Open(*args.input)
		if err != nil {
			logrus.Fatal(err)
		}
		defer func() {
			_ = f.Close()
		}()

		r, err = pdf.PlainText(slurper, pdf.PageRange{Start: 50, End: 51})
		if err != nil {
			logrus.Fatal(err)
		}
	} else {
		var err error
		r, err = os.Open(*args.input)
		if err != nil {
			logrus.Fatal(err)
		}
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		logrus.Fatal(err)
	}

	doc, err := prose.NewDocument(string(b),
		prose.UsingTokenizer(prose.NewIterTokenizer(prose.UsingSanitizer(strings.NewReplacer("-\n", "")))),
	)
	if err != nil {
		logrus.Fatal(err)
	}

	if *args.tags {
		tagCounts := make(map[string]int)
		tagTextCounts := make(map[string]map[string]int)
		tags := make([]string, 0)
		for _, tok := range doc.Tokens() {
			//fmt.Println(tok.Text, tok.Tag, tok.Label)
			if _, ok := tagCounts[tok.Tag]; ok {
				tagCounts[tok.Tag]++
			} else {
				tags = append(tags, tok.Tag)
				tagCounts[tok.Tag] = 1
			}
			text := strings.ToLower(tok.Text)
			if _, ok := tagTextCounts[tok.Tag]; ok {
				if _, ok := tagTextCounts[tok.Tag][text]; ok {
					tagTextCounts[tok.Tag][text]++
				} else {
					tagTextCounts[tok.Tag][text] = 1
				}
			} else {
				tagTextCounts[tok.Tag] = map[string]int{text: 1}
			}
		}

		sort.Strings(tags)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		_, _ = fmt.Fprintf(w, "Tag\tDescription\tTotal\tValues\n")
		for _, tag := range tags {
			var textCounts []textCount
			for k, v := range tagTextCounts[tag] {
				textCounts = append(textCounts, textCount{
					text: k, count: v,
				})
			}
			sort.Slice(textCounts, func(i, j int) bool {
				return textCounts[i].count > textCounts[j].count
			})
			_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%v\n", tag, tagDescriptions[tag], tagCounts[tag], textCounts)
		}
		_ = w.Flush()
	}

	if *args.entities {
		// Iterate over the doc's named-entities:
		for _, ent := range doc.Entities() {
			fmt.Println(ent.Text, ent.Label)
		}
	}
	if *args.sentences {
		// Iterate over the doc's sentences:
		for _, sent := range doc.Sentences() {
			fmt.Println(sent.Text)
		}
	}
}

type textCount struct {
	text  string
	count int
}
