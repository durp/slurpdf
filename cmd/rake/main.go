package main

import (
	"fmt"
	rake "github.com/afjoseph/RAKE.Go"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"os"
	"slurpdf/internal/pdf"
)

var (
	app  = kingpin.New("rake", "consume a pdf file and create a summary of the digested tokens")
	args = struct {
		input *string
	}{
		input: app.Flag("in", "input file to process").Short('i').Required().String(),
	}
)

func main() {
	kingpin.MustParse(app.Parse(os.Args[1:]))
	logrus.Infof("input: %s", *args.input)
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

	candidates := rake.RunRake(string(b))
	for _, candidate := range candidates {
		fmt.Printf("%s (%f)\n", candidate.Key, candidate.Value)
		if candidate.Value == 1.0 {
			break
		}
	}

}
