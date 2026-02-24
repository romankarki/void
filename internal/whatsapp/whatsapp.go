package whatsapp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Client struct {
	client *whatsmeow.Client
	store  *sqlstore.Container
}

type Chat struct {
	JID    string
	Name   string
	Last   string
	Time   time.Time
	Unread int
}

type Message struct {
	Sender  string
	Content string
	Time    time.Time
	IsMe    bool
}

func NewClient() (*Client, error) {
	dbPath := os.Getenv("USERPROFILE") + "\\.void\\whatsapp.db"

	container, err := sqlstore.New(context.Background(), "sqlite3", "file:"+dbPath+"?_foreign_keys=on", waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("no device found: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, waLog.Noop)

	return &Client{
		client: client,
		store:  container,
	}, nil
}

func (c *Client) Connect(onQR func(string), onConnected func()) error {
	c.client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.QR:
			qrData := v.Codes[len(v.Codes)-1]
			onQR(qrData)
		case *events.Connected:
			onConnected()
		case *events.Message:
			c.handleMessage(v)
		}
	})

	return c.client.Connect()
}

func (c *Client) handleMessage(evt *events.Message) {
	msg := evt.Message
	if msg.Conversation != nil || msg.ExtendedTextMessage != nil {
		var text string
		if msg.Conversation != nil {
			text = *msg.Conversation
		} else if msg.ExtendedTextMessage != nil {
			text = *msg.ExtendedTextMessage.Text
		}

		sender := evt.Info.Sender.String()
		if sender == "" {
			sender = evt.Info.Chat.String()
		}

		fmt.Printf("\n\x1b[32m[WhatsApp] %s: %s\x1b[0m\n", sender, text)
	}
}

func (c *Client) Disconnect() {
	c.client.Disconnect()
	c.store.Close()
}

func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

func (c *Client) GetChats() ([]Chat, error) {
	return []Chat{}, nil
}

func (c *Client) GetMessages(jid string, count int) ([]Message, error) {
	return []Message{}, nil
}

func (c *Client) SendMessage(jid string, text string) error {
	parsedJID, err := types.ParseJID(jid)
	if err != nil {
		return err
	}

	_, err = c.client.SendMessage(context.Background(), parsedJID, &waProto.Message{
		Conversation: &text,
	})
	return err
}

func PrintQR(qrData string) {
	fmt.Print("\x1b[2J\x1b[H")
	fmt.Println("\n \x1b[1;36m WhatsApp Login\x1b[0m")
	fmt.Println(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m")
	fmt.Println(" \x1b[90mScan the QR code with WhatsApp on your phone\x1b[0m")
	fmt.Println(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n")

	qrterminal.GenerateHalfBlock(qrData, qrterminal.L, os.Stdout)

	fmt.Println("\n \x1b[90mOpen WhatsApp → Settings → Linked Devices → Link a Device\x1b[0m")
}

func (c *Client) GetContactName(jid string) string {
	return jid
}

func (c *Client) Logout() error {
	return c.client.Logout(context.Background())
}

func (c *Client) GetQRChannel() (<-chan whatsmeow.QRChannelItem, error) {
	return c.client.GetQRChannel(context.Background())
}

func FormatChats(chats []Chat) string {
	var sb strings.Builder
	sb.WriteString("\n \x1b[1;36m WhatsApp Chats\x1b[0m\n")
	sb.WriteString(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n")

	for i, chat := range chats {
		if i >= 15 {
			break
		}

		unread := ""
		if chat.Unread > 0 {
			unread = fmt.Sprintf(" \x1b[1;32m(%d)\x1b[0m", chat.Unread)
		}

		last := chat.Last
		if len(last) > 30 {
			last = last[:27] + "..."
		}

		sb.WriteString(fmt.Sprintf(" \x1b[1;33m%-20s\x1b[0m %s%s\n", chat.Name, last, unread))
	}

	return sb.String()
}

func FormatMessages(messages []Message, myJID string) string {
	var sb strings.Builder
	sb.WriteString(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n")

	for _, msg := range messages {
		timeStr := msg.Time.Format("15:04")
		if msg.IsMe {
			sb.WriteString(fmt.Sprintf(" \x1b[90m%s\x1b[0m \x1b[32m%s\x1b[0m: %s\n", timeStr, "Me", msg.Content))
		} else {
			sb.WriteString(fmt.Sprintf(" \x1b[90m%s\x1b[0m \x1b[33m%s\x1b[0m: %s\n", timeStr, msg.Sender, msg.Content))
		}
	}

	return sb.String()
}
