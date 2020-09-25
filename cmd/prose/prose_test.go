package main

import (
	"fmt"
	"github.com/jdkato/prose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestProseHyphenation(t *testing.T) {
	doc, err := prose.NewDocument("This was a hyphen-\nated sentence! This is not a hyphenated sentence.",
		prose.UsingTokenizer(prose.NewIterTokenizer(prose.UsingSanitizer(strings.NewReplacer("-\n","")))))
	require.NoError(t, err)
	sentences := doc.Sentences()
	assert.Equal(t, "This was a hyphen-\nated sentence!", sentences[0].Text)
	assert.Equal(t, "This is not a hyphenated sentence.", sentences[1].Text)
	tokens := doc.Tokens()
	for _, token := range tokens {
		fmt.Println(token)
	}
	assert.Equal(t, "hyphenated", tokens[3].Text)

}
