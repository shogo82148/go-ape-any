package slack

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/shogo82148/go-ape-any"
)

type Slack struct {
	RTM           *slack.RTM
	API           *slack.Client
	bot           ape.Handler
	myName        string // my nickname
	myEncodedName string // <@Uxxxxx> encoded name
	done          chan struct{}
}

func New(bot ape.Handler, token string) *Slack {
	return &Slack{
		API:  slack.New(token),
		bot:  bot,
		done: make(chan struct{}, 1),
	}
}

func (p *Slack) Send(to, message string) error {
	p.API.PostMessage(to, message, slack.PostMessageParameters{AsUser: true})
	return nil
}

func (p *Slack) Run() error {
	startTime := time.Now().Unix()

	p.RTM = p.API.NewRTM()
	go p.RTM.ManageConnection()

	for {
		select {
		case msg := <-p.RTM.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				p.myName = p.RTM.GetInfo().User.Name
				p.myEncodedName = "<@" + p.RTM.GetInfo().User.ID + ">"
			case *slack.MessageEvent:
				if f, err := strconv.ParseFloat(ev.Timestamp, 64); err == nil {
					if int64(f) < startTime {
						continue
					}
				}
				if ev.SubType == "" || ev.SubType == "bot_message" {
					go p.bot.HandleEvent(p.newEvent(ev), nil)
				}
			case *slack.InvalidAuthEvent:
				return errors.New("invalid auth")
			}
		case <-p.done:
			return nil
		}
	}
}

func (p *Slack) Stop() error {
	p.done <- struct{}{}
	return nil
}

var regexpSpace = regexp.MustCompile(`\s+`)

func (p *Slack) newEvent(ev *slack.MessageEvent) *Event {
	e := &Event{ev: ev, slack: p}
	text := e.ev.Text

	// find reply-to taget name
	if colon := strings.Index(text, ":"); colon > 0 {
		targetName := strings.TrimSpace(text[:colon])
		if strings.EqualFold(targetName, p.myName) || strings.EqualFold(targetName, p.myEncodedName) {
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
	ev    *slack.MessageEvent
	slack *Slack

	text        string
	command     string
	args        []string
	isReplyToMe bool
}

func (e *Event) Command() string        { return e.command }
func (e *Event) Args() []string         { return e.args }
func (e *Event) Channel() string        { return e.slack.RTM.GetInfo().GetChannelByID(e.ev.Channel).Name }
func (e *Event) Text() string           { return e.text }
func (e *Event) Nick() string           { return e.slack.RTM.GetInfo().GetUserByID(e.ev.User).Name }
func (e *Event) IsReplyToMe() bool      { return e.isReplyToMe }
func (e *Event) Provider() ape.Provider { return e.slack }

func (e *Event) Reply(message string) error {
	rtm := e.slack.RTM
	rtm.SendMessage(rtm.NewOutgoingMessage(message, e.ev.Channel))
	return nil
}

func (e *Event) Typing() error {
	rtm := e.slack.RTM
	rtm.SendMessage(rtm.NewTypingMessage(e.ev.Channel))
	return nil
}
