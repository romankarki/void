package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mdp/qrterminal"
	_ "modernc.org/sqlite"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	waProto "go.mau.fi/whatsmeow/binary/proto"
)

type Client struct {
	client      *whatsmeow.Client
	store       *sqlstore.Container
	messageChan chan *events.Message
	connected   bool
	myJID       string
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
	configDir := os.Getenv("USERPROFILE") + string(os.PathSeparator) + ".void"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	dbPath := configDir + string(os.PathSeparator) + "whatsapp.db"

	// Open with foreign keys enabled
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	db.Close()

	// Use the same connection string for sqlstore
	container, err := sqlstore.New(context.Background(), "sqlite", dbPath+"?_pragma=foreign_keys(ON)", waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		deviceStore = container.NewDevice()
	}

	client := whatsmeow.NewClient(deviceStore, waLog.Noop)

	return &Client{
		client:      client,
		store:       container,
		messageChan: make(chan *events.Message, 100),
		connected:   false,
	}, nil
}

func (c *Client) Connect(onQR func(string), onConnected func()) error {
	c.client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.QR:
			if len(v.Codes) > 0 {
				onQR(v.Codes[len(v.Codes)-1])
			}
		case *events.Connected:
			c.connected = true
			if c.client.Store.ID != nil {
				c.myJID = c.client.Store.ID.String()
			}
			onConnected()
		case *events.Disconnected:
			c.connected = false
		case *events.Message:
			select {
			case c.messageChan <- v:
			default:
			}
			c.printMessage(v)
		}
	})

	return c.client.Connect()
}

func (c *Client) printMessage(evt *events.Message) {
	msg := evt.Message
	var text string

	if msg.Conversation != nil {
		text = *msg.Conversation
	} else if msg.ExtendedTextMessage != nil && msg.ExtendedTextMessage.Text != nil {
		text = *msg.ExtendedTextMessage.Text
	} else if msg.ImageMessage != nil && msg.ImageMessage.Caption != nil {
		text = "[Image] " + *msg.ImageMessage.Caption
	} else if msg.VideoMessage != nil && msg.VideoMessage.Caption != nil {
		text = "[Video] " + *msg.VideoMessage.Caption
	} else if msg.AudioMessage != nil {
		text = "[Audio message]"
	}

	if text == "" {
		return
	}

	sender := evt.Info.Sender.User
	if sender == "" {
		sender = evt.Info.Chat.User
	}

	fmt.Printf("\n\x1b[2K\r\x1b[1;32m[%s]\x1b[0m \x1b[1;33m%s:\x1b[0m %s\n",
		time.Now().Format("15:04"), sender, text)
}

func (c *Client) Disconnect() {
	c.client.Disconnect()
	c.store.Close()
}

func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

func (c *Client) SendMessage(jid string, text string) error {
	parsedJID, err := types.ParseJID(jid)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	_, err = c.client.SendMessage(context.Background(), parsedJID, &waProto.Message{
		Conversation: &text,
	})
	return err
}

func (c *Client) SendMessageToPhone(phone string, text string) error {
	jid := types.NewJID(phone, types.DefaultUserServer)
	return c.SendMessage(jid.String(), text)
}

func (c *Client) Logout() error {
	return c.client.Logout(context.Background())
}

func (c *Client) GetMessageChannel() <-chan *events.Message {
	return c.messageChan
}

func PrintQR(qrData string) {
	fmt.Print("\x1b[2J\x1b[H")
	fmt.Println("\n \x1b[1;36m============================================\x1b[0m")
	fmt.Println(" \x1b[1;36m  WhatsApp Login\x1b[0m")
	fmt.Println(" \x1b[36m============================================\x1b[0m")
	fmt.Println(" \x1b[90mScan with WhatsApp > Settings > Linked Devices\x1b[0m")
	fmt.Println(" \x1b[36m============================================\x1b[0m\n")

	qrterminal.GenerateHalfBlock(qrData, qrterminal.L, os.Stdout)
}

type TerminalUI struct {
	client      *Client
	currentChat string
}

func NewTerminalUI(client *Client) *TerminalUI {
	return &TerminalUI{client: client}
}

func (ui *TerminalUI) Run() int {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Print("\x1b[2J\x1b[H")
	ui.renderHelp()

	inputChan := make(chan string)
	go ui.readInput(inputChan)

	for {
		select {
		case <-sigChan:
			fmt.Println("\n\n \x1b[90mDisconnecting...\x1b[0m")
			ui.client.Disconnect()
			return 0
		case input := <-inputChan:
			if strings.TrimSpace(input) == "" {
				continue
			}
			ui.handleCommand(input)
		}
	}
}

func (ui *TerminalUI) readInput(ch chan<- string) {
	var buf [256]byte
	var pos int

	for {
		n, err := os.Stdin.Read(buf[pos : pos+1])
		if err != nil || n == 0 {
			continue
		}

		if buf[pos] == '\r' || buf[pos] == '\n' {
			line := string(buf[:pos])
			pos = 0
			if line != "" {
				ch <- line
			}
			fmt.Print("\n\x1b[1;36m>\x1b[0m ")
		} else if buf[pos] == 127 || buf[pos] == 8 {
			if pos > 0 {
				pos--
				fmt.Print("\b \b")
			}
		} else if buf[pos] >= 32 {
			fmt.Print(string(buf[pos : pos+1]))
			pos++
			if pos >= len(buf) {
				pos = 0
			}
		}
	}
}

func (ui *TerminalUI) handleCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "send", "s":
		if len(args) < 2 {
			fmt.Println(" \x1b[31mUsage: send <phone> <message>\x1b[0m")
			break
		}
		phone := args[0]
		message := strings.Join(args[1:], " ")
		if err := ui.client.SendMessageToPhone(phone, message); err != nil {
			fmt.Printf(" \x1b[31mError: %v\x1b[0m\n", err)
		} else {
			fmt.Println(" \x1b[32mMessage sent!\x1b[0m")
		}
	case "chat", "c":
		if len(args) < 1 {
			fmt.Println(" \x1b[31mUsage: chat <phone>\x1b[0m")
			break
		}
		ui.currentChat = args[0] + "@s.whatsapp.net"
		fmt.Printf(" \x1b[32mChat: %s\x1b[0m\n", args[0])
	case "msg", "m":
		if ui.currentChat == "" {
			fmt.Println(" \x1b[31mNo chat. Use: chat <phone>\x1b[0m")
			break
		}
		message := strings.Join(args, " ")
		if err := ui.client.SendMessage(ui.currentChat, message); err != nil {
			fmt.Printf(" \x1b[31mError: %v\x1b[0m\n", err)
		} else {
			fmt.Println(" \x1b[32mSent!\x1b[0m")
		}
	case "help", "h":
		ui.renderHelp()
	case "quit", "q", "exit":
		ui.client.Disconnect()
		os.Exit(0)
	case "clear":
		fmt.Print("\x1b[2J\x1b[H")
		ui.renderHelp()
	default:
		fmt.Printf(" \x1b[31mUnknown: %s\x1b[0m\n", cmd)
	}

	fmt.Print("\n\x1b[1;36m>\x1b[0m ")
}

func (ui *TerminalUI) renderHelp() {
	fmt.Println(" \x1b[1;36m============================================\x1b[0m")
	fmt.Println(" \x1b[1;36m  WhatsApp CLI - Connected!\x1b[0m")
	fmt.Println(" \x1b[36m============================================\x1b[0m")
	fmt.Println()
	fmt.Println(" \x1b[1;33mCommands:\x1b[0m")
	fmt.Println("   send <phone> <msg>  - Send to phone")
	fmt.Println("   chat <phone>        - Start chat")
	fmt.Println("   msg <text>          - Send in chat")
	fmt.Println("   clear               - Clear screen")
	fmt.Println("   help                - Show help")
	fmt.Println("   exit                - Quit")
	fmt.Println()
	fmt.Println(" \x1b[90mPhone: country code + number (e.g., 9779800000000)\x1b[0m")
	fmt.Println(" \x1b[36m============================================\x1b[0m")
}
