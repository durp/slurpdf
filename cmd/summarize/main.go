package main

import (
	"errors"
	"fmt"
	textrank "github.com/DavidBelicza/TextRank"
	"github.com/DavidBelicza/TextRank/rank"
	"github.com/JesusIslam/tldr"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"os"
	"slurpdf/internal/pdf"
)

var (
	app  = kingpin.New("sum", "consume a pdf file and create a summary")
	args = struct {
		input  *string
		sumLen *int
	}{
		input:  app.Flag("in", "input file to process").Short('i').Required().String(),
		sumLen: app.Flag("sumlen", "number of summary sentences to return").Short('n').Default("5").Int(),
	}
	lexRank = app.Command("lexrank", "use LexRank to summarize")
	textRank = app.Command("textrank", "use TextRank to summarize")
	textArgs = struct {
		method *string
	}{
		method: textRank.Flag("method", "`qty` for quantity or `rel` for relationship").Short('m').String(),
	}
)

func main() {
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	logrus.Infof("input: %s", *args.input)

	var rankFn func([]byte, int) ([]string, error)
	switch cmd {
	case lexRank.FullCommand():
		rankFn = lex
	case textRank.FullCommand():
		rankFn = text(*textArgs.method)
	}

	f, slurper, err := pdf.Open(*args.input)
	if err != nil {
		logrus.Fatal(err)
	}
	defer func() {
		_ = f.Close()
	}()

	r, err := pdf.PlainText(slurper)
	if err != nil {
		logrus.Fatal(err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		logrus.Fatal(err)
	}

	sum, err := rankFn(b, *args.sumLen)
	if err != nil {
		logrus.Fatal(err)
	}
	for i, s := range sum {
		fmt.Printf("[%d] %s\n", i+1, s)
	}
}

func lex(b []byte, sumLength int) ([]string, error) {
	bag := tldr.New()
	return bag.Summarize(string(b), sumLength)
}

func text(method string) func(b []byte, sumLength int) ([]string, error) {
	return func(b []byte, sumLength int) ([]string, error) {
		var sumFn func(*textrank.TextRank, int) []rank.Sentence
		switch method {
		case "qty":
			sumFn =  textrank.FindSentencesByWordQtyWeight
		case "rel":
			sumFn =  textrank.FindSentencesByRelationWeight
		default:
			return []string{}, errors.New("invalid method, only `qty` and `rel` supported")
		}

		tr := textrank.NewTextRank()
		tr.Populate(string(b), textrank.NewDefaultLanguage(), textrank.NewDefaultRule())
		tr.Ranking(textrank.NewDefaultAlgorithm())

		sentences := sumFn(tr, sumLength)
		result := make([]string,0,sumLength)
		for i := 0; i < sumLength; i++ {
			result = append(result, sentences[i].Value)
		}
		return result, nil
	}
}
