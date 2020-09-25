package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"slurpdf/internal/pdf"
	"sort"
	"unicode"
)

var (
	app  = kingpin.New("bagowords", "consume a pdf file and create a summary of the digested tokens")
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

	scanner := bufio.NewScanner(r)
	cleaner := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = bufio.ScanWords(data, atEOF)
		token = bytes.ToLower(token)
		if err != nil {
			return
		}
		token = bytes.TrimFunc(token, unicode.IsPunct)
		for _, p := range particles() {
			if bytes.Equal([]byte(p), token) {
				return advance, nil, nil
			}
		}

		return
	}

	scanner.Split(cleaner)

	// This is pretty awful, we have just slurped the whole damn thing in and now we are going at it again to tokenize
	bag := make(map[string]int,1000)
	for scanner.Scan() {
		tok := scanner.Text()
		bag[tok]++
	}

	type kv struct {
		Key string
		Value int
	}

	var byv []kv
	for k, v := range bag {
		byv = append(byv, kv{k, v})
	}

	sort.Slice(byv, func(i, j int) bool {
		return byv[i].Value > byv[j].Value
	})

	for _, kv := range byv {
		fmt.Printf("%s %d\n", kv.Key, kv.Value)
	}

	fmt.Printf("%+v", bag)
}

func particles() []string {
	return []string {
		"the", "of", "and", "a", "to", "is", "in", "or", "for", "be", "may", "are", "as", "on", "with", "by", "not", "one",
		"that", "at", "an", "has", "if", "he", "each", "it", "can", "such", "this", "his", "will", "use", "any", "all", "from",
		"no", "per", "they", "but", "their", "who", "during", "should", "only", "using", "she", "than", "once", "into", "been",
		"being", "does", "then", "thus", "between", "do", "other", "used", "where", "some", "also",
	}
}