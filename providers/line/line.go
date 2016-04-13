package line

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"

	"github.com/shogo82148/go-ape-any"
)

type Line struct {
	bot           ape.Handler
	channelID     string
	channelSecret string
	mid           string
	address       string
	listener      net.Listener
}

type ContentType int

const (
	TypeText     ContentType = 1
	TypeImage                = 2
	TypeVideo                = 3
	TypeAudio                = 4
	TypeLocation             = 7
	TypeSticker              = 8
	TypeContact              = 10
)

type ReceivingMessage struct {
	From        string            `json:"from"`
	FromChannel string            `json:"fromChannel"`
	To          []string          `json:"to"`
	EventType   string            `json:"eventType"`
	ID          string            `json:"id"`
	Content     *ReceivingContent `json:"content"`
}

type ReceivingContent struct {
	json.RawMessage
	contentType ContentType
}

func (rc *ReceivingContent) ContentType() ContentType {
	if rc.contentType != 0 {
		return rc.contentType
	}

	return rc.contentType
}

type SendingMessage struct {
	To        []string    `json:"to"`
	ToChannel int64       `json:"toChannel"`
	EventType string      `json:"eventType"`
	Content   interface{} `json:"content"`
}

type SendingTextContent struct {
	ContentType ContentType `json:"contentType"`
	ToType      int         `json:"toType"`
	Text        string      `json:"text"`
}

func New(bot ape.Handler, channelID, channelSecret, mid, address string) *Line {
	return &Line{
		bot:           bot,
		channelID:     channelID,
		channelSecret: channelSecret,
		mid:           mid,
	}
}

func (p *Line) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (p *Line) Send(to, message string) error {
	msg := &SendingMessage{
		To:        []string{to},
		ToChannel: 1383378250,
		EventType: "138311608800106203",
		Content: &SendingTextContent{
			ContentType: TypeText,
			ToType:      1,
			Text:        message,
		},
	}
	buf := &bytes.Buffer{}
	e := json.NewEncoder(buf)
	if err := e.Encode(msg); err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		"https://trialbot-api.line.me/v1/events",
		buf,
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charser=UTF-8")
	req.Header.Set("X-Line-ChannelID", p.channelID)
	req.Header.Set("X-Line-ChannelSecret", p.channelSecret)
	req.Header.Set("X-Line-Trusted-User-With-ACL", p.mid)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (p *Line) Run() error {
	ln, err := net.Listen("tcp", p.address)
	if err != nil {
		return err
	}
	p.listener = ln
	return http.Serve(ln, p)
}

func (p *Line) Stop() error {
	return p.listener.Close()
}

type Event struct {
	line *Line

	text    string
	command string
	args    []string
}

func (e *Event) Command() string        { return e.command }
func (e *Event) Args() []string         { return e.args }
func (e *Event) Channel() string        { return "" }
func (e *Event) Text() string           { return e.text }
func (e *Event) Nick() string           { return "" }
func (e *Event) IsReplyToMe() bool      { return true }
func (e *Event) Provider() ape.Provider { return e.line }

func (e *Event) Reply(message string) error {
	return nil
}
