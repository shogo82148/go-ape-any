package line

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

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
	TypeUnknown  ContentType = 0
	TypeText                 = 1
	TypeImage                = 2
	TypeVideo                = 3
	TypeAudio                = 4
	TypeLocation             = 7
	TypeSticker              = 8
	TypeContact              = 10
)

type ReceivingBody struct {
	Result []*ReceivingMessage `json:"result"`
}

type ReceivingMessage struct {
	From        string            `json:"from"`
	FromChannel int64             `json:"fromChannel"`
	To          []string          `json:"to"`
	ToChannel   int64             `json:"toChannel"`
	EventType   string            `json:"eventType"`
	ID          string            `json:"id"`
	CreatedTime int64             `json:"createdTime"`
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

	var t struct {
		ContentType ContentType `json:"contentType"`
	}
	if err := json.Unmarshal(rc.RawMessage, &t); err != nil {
		return TypeUnknown
	}
	rc.contentType = t.ContentType

	return rc.contentType
}

type ReceivingContentText struct {
	ID              string            `json:"id,omitempty"`
	From            string            `json:"from,omitempty"`
	CreatedTime     int64             `json:"createdTime,omitempty"`
	To              []string          `json:"to,omitempty"`
	ToType          int               `json:"toType"`
	ContentMetadata map[string]string `json:"contentMetadata,omitempty"`
	Text            string            `json:"text"`
	Location        map[string]string `json:"location,omitempty"`
}

func (rc *ReceivingContent) Text() (*ReceivingContentText, error) {
	t := &ReceivingContentText{}
	if err := json.Unmarshal(rc.RawMessage, t); err != nil {
		return nil, err
	}
	return t, nil
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
		address:       address,
	}
}

func (p *Line) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(r.Body); err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
	data := buf.Bytes()
	signature := r.Header.Get("X-Line-Channelsignature")
	go p.handleMessages(data, signature)
}

func (p *Line) handleMessages(data []byte, signature string) {
	// check signature
	hash := hmac.New(sha256.New, []byte(p.channelSecret))
	hash.Write(data)
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		log.Println("invalid signature:", signature)
	}
	if !hmac.Equal(signatureBytes, hash.Sum(nil)) {
		log.Println("signature check failed")
		return
	}

	log.Println(string(data))
	body := &ReceivingBody{}
	if err := json.Unmarshal(data, body); err != nil {
		log.Println(err)
		return
	}
	for _, message := range body.Result {
		go p.handleMessage(message)
	}
}

var regexpSpace = regexp.MustCompile(`\s+`)

func (p *Line) handleMessage(message *ReceivingMessage) {
	switch message.Content.ContentType() {
	case TypeText:
		text, err := message.Content.Text()
		if err != nil {
			log.Println(err)
			return
		}
		text.Text = strings.TrimSpace(text.Text)
		command := ""
		args := regexpSpace.Split(text.Text, -1)
		if len(args) > 0 {
			command = args[0]
			args = args[1:]
		}

		p.bot.HandleEvent(&Event{
			line:    p,
			from:    text.From,
			text:    text.Text,
			command: command,
			args:    args,
		}, nil)
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

	from    string
	text    string
	command string
	args    []string
}

func (e *Event) Command() string        { return e.command }
func (e *Event) Args() []string         { return e.args }
func (e *Event) Channel() string        { return e.from }
func (e *Event) Text() string           { return e.text }
func (e *Event) Nick() string           { return "" }
func (e *Event) IsReplyToMe() bool      { return true }
func (e *Event) Provider() ape.Provider { return e.line }

func (e *Event) Reply(message string) error {
	return e.line.Send(e.from, message)
}
