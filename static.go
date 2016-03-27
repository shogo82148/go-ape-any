package ape

import "regexp"

var defaultBot *Bot = New()

func HandleMessage(r *regexp.Regexp, h Handler) {
	defaultBot.HandleMessage(r, h)
}

func HandleMessageFunc(r *regexp.Regexp, f func(e Event, args []string)) {
	defaultBot.HandleMessageFunc(r, f)
}

func Handle(name string, h Handler) {
	defaultBot.Handle(name, h)
}

func HandleFunc(name string, f func(e Event, args []string)) {
	defaultBot.HandleFunc(name)
}

func HandleDefault(h Handler) {
	defaultBot.HandleDefault(h)
}

func (bot *Bot) HandleDefaultFunc(f func(e Event, args []string)) {
	defaultBot.HandleDefaultFunc(f)
}
