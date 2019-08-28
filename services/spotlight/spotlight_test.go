package spotlight

import (
	"testing"

	"github.com/stretchr/testify/assert"
	stripmd "github.com/writeas/go-strip-markdown"
)

func TestShortText(t *testing.T) {
	text := "Hello"
	lines := wordWrap(text, 7)
	assert.Equal(t, len(lines), 1)
	assert.Equal(t, lines[0], text)
}

func TestLongText(t *testing.T) {
	text := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Maecenas ultricies leo at metus porta, sed fermentum nibh malesuada. Donec vehicula ligula ut turpis efficitur gravida. Proin mattis aliquet pharetra. Curabitur vitae elit purus. Etiam aliquet metus ac neque rhoncus, non commodo arcu blandit. Pellentesque in ultricies magna. Nulla nec felis."
	lines := wordWrap(text, 7)
	assert.Equal(t, len(lines) > 1, true)
}

func TestMarkdownText(t *testing.T) {
	text := "## Heading"
	lines := wordWrap(text, 7)
	assert.Equal(t, len(lines), 1)
	assert.Equal(t, lines[0], stripmd.Strip(text))
}

func TestMentionText(t *testing.T) {
	text := "I agree with @cosmos1xqc5gsesg5m4jv252ce9g4jgfev52s68an2ss9."
	lines := wordWrap(text, 7)
	assert.Equal(t, len(lines), 1)
	assert.Equal(t, lines[0], "I agree with @cosmos1xqc...2ss9.")
}

func TestLinkText(t *testing.T) {
	text := "The link http://someveryveryverylongurlgoeshere.com/and-the-url-doesnt-seem-to-end-anytime-soon/what-would-happen-now?id=123 says that TruStory is awesome."
	lines := wordWrap(text, 7)
	assert.Equal(t, len(lines), 3)
	assert.Equal(t, lines[0], "The link")
	assert.Equal(t, lines[1], "http://someveryveryv... says that")
	assert.Equal(t, lines[2], "TruStory is awesome.")
}
