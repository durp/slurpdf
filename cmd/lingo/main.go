package main

import (
	"fmt"
	"github.com/chewxy/lingo/dep"
	"github.com/chewxy/lingo/lexer"
	"github.com/chewxy/lingo/pos"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"strings"
)

var (
	app  = kingpin.New("lingo", "consume a pdf file and create a summary of the digested tokens")
	args = struct {
		input *string
		posModel *string
		depModel *string
	}{
		input: app.Flag("in", "input file to process").Short('i').Required().String(),
		posModel: app.Flag("pos", "pos model to use").Short('p').Required().String(),
		depModel: app.Flag("dep", "dep model to use").Short('d').Required().String(),
	}
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))
	logrus.Infof("input: %s", *args.input)
	//f, slurper, err := pdf.Open(*args.input)
	//if err != nil {
	//	logrus.Fatal(err)
	//}
	//defer func() {
	//	_ = f.Close()
	//}()
	//
	//r, err := pdf.PlainText(slurper)
	//if err != nil {
	//	logrus.Fatal(err)
	//}

	r := strings.NewReader("The quick brown fox jumps over the lazy dog.")

	lx := lexer.New("?", r) // lexer - required to break a sentence up into words.
	pmod, err := pos.Load(*args.posModel)
	if err != nil {
		logrus.Fatal(err)
	}
	pt := pos.New(pos.WithModel(pmod)) // POS Tagger - required to tag the words with a part of speech tag.

	dmod, err := dep.Load(*args.depModel)
	if err != nil {
		logrus.Fatal(err)
	}
	dp := dep.New(dmod)                // Creates a new parser
	fmt.Printf("%s", dp)

	// set up a pipeline
	pt.Input = lx.Output
	dp.Input = pt.Output

	// run all
	go lx.Run()
	go pt.Run()
	go dp.Run()

	// wait to receive:
	for {
		select {
		case d := <-dp.Output:
			if d != nil {
				fmt.Printf("%v\n", d.Tree().Dot())
			}
		case err := <-dp.Error:
			logrus.Error(err)
		}
	}

}
