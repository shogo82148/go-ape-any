package ape

import "regexp"

type Provider interface {
	Send(to, message string) error
}

type Event interface {
	// Command is the name of the command invoked.
	Command() string

	// Args is the args of the command invoked.
	Args() []string

	// Channel is the channel name
	Channel() string

	// Text is the message text spoken (excluding reply-to).
	Text() string

	// Nick is the nickname of the user speaking.
	Nick() string

	// Reply sends a message to event source.
	Reply(message string) error

	// IsReplyToMe return if the message replied to me.
	IsReplyToMe() bool

	// Provider is the provider of event
	Provider() Provider
}

type Handler interface {
	HandleEvent(e Event, extraArgs []string)
}

type HandlerFunc func(e Event, extrArgs []string)

func (f HandlerFunc) HandleEvent(e Event, args []string) {
	f(e, args)
}

type regexpEntry struct {
	regexp  *regexp.Regexp
	handler Handler
}

type Bot struct {
	regexps       []regexpEntry
	actions       map[string]Handler
	defaultAction Handler
}

func (bot *Bot) HandleEvent(e Event, args []string) {
	for _, r := range bot.regexps {
		if match := r.regexp.FindStringSubmatch(e.Text()); match != nil {
			r.handler.HandleEvent(e, match)
			return
		}
	}

	if e.IsReplyToMe() {
		name := e.Command()
		action, ok := bot.actions[name]
		if !ok {
			action = bot.defaultAction
		}
		if action != nil {
			action.HandleEvent(e, e.Args())
		}
	}
}

func (bot *Bot) HandleMessage(r *regexp.Regexp, h Handler) {
	bot.regexps = append(bot.regexps, regexpEntry{r, h})
}

func (bot *Bot) HandleMessageFunc(r *regexp.Regexp, f func(e Event, args []string)) {
	bot.HandleMessage(r, HandlerFunc(f))
}

func (bot *Bot) Handle(name string, h Handler) {
	bot.actions[name] = h
}

func (bot *Bot) HandleFunc(name string, f func(e Event, args []string)) {
	bot.actions[name] = HandlerFunc(f)
}

func (bot *Bot) HandleDefault(h Handler) {
	bot.defaultAction = h
}

func (bot *Bot) HandleDefaultFunc(f func(e Event, args []string)) {
	bot.defaultAction = HandlerFunc(f)
}

func New() *Bot {
	return &Bot{
		actions: map[string]Handler{},
	}
}
