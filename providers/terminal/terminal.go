package terminal

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/shogo82148/go-ape-any"
	"golang.org/x/crypto/ssh/terminal"
)

type Terminal struct {
	bot    ape.Handler
	myName string // my nickname
	term   *terminal.Terminal
}

func New(bot ape.Handler, myName string) *Terminal {
	return &Terminal{
		bot:    bot,
		myName: myName,
	}
}

func (p *Terminal) Send(to, message string) error {
	fmt.Fprintln(p.term, to+": "+message)
	return nil
}

func (p *Terminal) Run() error {
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		return err
	}
	defer terminal.Restore(0, oldState)

	term := terminal.NewTerminal(os.Stdin, "> ")
	p.term = term
	for {
		line, err := term.ReadLine()
		if err != nil {
			return err
		}
		go p.bot.HandleEvent(p.newEvent(line), nil)
	}
	return nil
}

var regexpSpace = regexp.MustCompile(`\s+`)

func (p *Terminal) newEvent(text string) *Event {
	e := &Event{terminal: p}

	// find reply-to taget name
	if colon := strings.Index(text, ":"); colon > 0 {
		targetName := strings.TrimSpace(text[:colon])
		if strings.EqualFold(targetName, p.myName) {
			e.isReplyToMe = true
			text = text[colon+1:]
		}
	}

	// parse command
	text = strings.TrimSpace(text)
	args := regexpSpace.Split(text, -1)
	if len(args) > 0 {
		e.command = args[0]
		e.args = args[1:]
	}
	e.text = text

	return e
}

type Event struct {
	terminal *Terminal

	text        string
	command     string
	args        []string
	isReplyToMe bool
}

func (e *Event) Command() string        { return e.command }
func (e *Event) Args() []string         { return e.args }
func (e *Event) Channel() string        { return "#stdin" }
func (e *Event) Text() string           { return e.text }
func (e *Event) Nick() string           { return "stdin" }
func (e *Event) IsReplyToMe() bool      { return e.isReplyToMe }
func (e *Event) Provider() ape.Provider { return e.terminal }

func (e *Event) Reply(message string) error {
	return e.terminal.Send(e.Channel(), message)
}
