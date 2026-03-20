package mods

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// QueueWaiter watches chat for queue position updates and reports them.
// It prints [QUEUE] position/total to stdout for the app to read.
// When position reaches the notify threshold it prints [QUEUE_READY].

type QueueWaiter struct {
	position      int
	total         int
	enabled       bool
	notifyAt      int
	lastPrint     time.Time
	sendToClient  func(packet.Packet) error
}

// Common queue message patterns across servers
var queuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(\d+)$`),
	regexp.MustCompile(`(?i)position in queue[:\s]+(\d+)`),
	regexp.MustCompile(`(?i)your queue position is (\d+)`),
	regexp.MustCompile(`(?i)you are (\d+) in queue`),
	regexp.MustCompile(`(?i)position[:\s]+#?(\d+)`),
	regexp.MustCompile(`(?i)queue[:\s]+#?(\d+)`),
	regexp.MustCompile(`(?i)you are #?(\d+) in (queue|line)`),
	regexp.MustCompile(`(?i)(\d+) player[s]? ahead`),
	regexp.MustCompile(`(?i)place[:\s]+#?(\d+)`),
	regexp.MustCompile(`(?i)#(\d+) in (queue|line)`),
	regexp.MustCompile(`(?i)(\d+)/(\d+) in queue`),
}

var totalPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(\d+)/(\d+)`),
	regexp.MustCompile(`(?i)of (\d+)`),
}

func init() { Register(&QueueWaiter{notifyAt: 10}) }

func (q *QueueWaiter) Name() string { return "QueueWaiter" }

func (q *QueueWaiter) Init(ctx *Context, _ any) {
	q.sendToClient = ctx.SendToClient
	fmt.Println("[QueueWaiter] ready - watching for queue messages")
}

func (q *QueueWaiter) Tick() {
	if !q.enabled || q.position == 0 {
		return
	}
	// Print queue status every 5 seconds
	if time.Since(q.lastPrint) > 5*time.Second {
		q.lastPrint = time.Now()
		if q.total > 0 {
			fmt.Printf("[QUEUE] %d/%d\n", q.position, q.total)
		} else {
			fmt.Printf("[QUEUE] %d\n", q.position)
		}
	}
}

func (q *QueueWaiter) HandlePacket(ctx *Context) {
	if ctx.Direction != FromServer {
		return
	}
	// Debug: log all packet types to find queue position
	var msg string
	switch pk := ctx.Packet.(type) {
	case *packet.Text:
		msg = strings.TrimSpace(pk.Message)
		if len(msg) < 100 { fmt.Printf("[QueueWaiter] text: %q\n", msg) }
	case *packet.SetTitle:
		msg = strings.TrimSpace(pk.Text)
		fmt.Printf("[QueueWaiter] title: %q\n", msg)
	default:
		return
	}
	if msg == "" {
		return
	}

	// Try to extract position
	pos := q.extractPosition(msg)
	if pos <= 0 {
		return
	}

	q.enabled = true
	oldPos := q.position
	q.position = pos

	// Try to extract total
	total := q.extractTotal(msg)
	if total > 0 {
		q.total = total
	}

	// Print update immediately
	if q.total > 0 {
		fmt.Printf("[QUEUE] %d/%d\n", q.position, q.total)
	} else {
		fmt.Printf("[QUEUE] %d\n", q.position)
	}

	// Show in game action bar
	if q.sendToClient != nil {
		var bar string
		if q.total > 0 {
			bar = fmt.Sprintf("\xa78[\xa7eQueue\xa78] \xa7fPosition: \xa7e%d\xa7f/\xa7e%d", q.position, q.total)
		} else {
			bar = fmt.Sprintf("\xa78[\xa7eQueue\xa78] \xa7fPosition: \xa7e%d", q.position)
		}
		_ = q.sendToClient(&packet.SetTitle{
			ActionType: packet.TitleActionSetActionBar,
			Text:       bar,
		})
	}

	// Notify when reaching threshold
	if oldPos > q.notifyAt && q.position <= q.notifyAt {
		fmt.Printf("[QUEUE_READY] %d\n", q.position)
		if q.sendToClient != nil {
			_ = q.sendToClient(&packet.Text{
				TextType: packet.TextTypeSystem,
				Message:  fmt.Sprintf("\xa7a[Queue] Almost in! Position: \xa7e%d\xa7a - Open Minecraft now!", q.position),
			})
		}
	}

	// Done when position is 1 or 0
	if q.position <= 1 {
		fmt.Println("[QUEUE_READY] 1")
		q.enabled = false
		q.position = 0
		q.total = 0
	}
}

func stripFormatting(msg string) string {
re := regexp.MustCompile(`§[0-9a-fk-or]`)
return strings.TrimSpace(re.ReplaceAllString(msg, ""))
}

func (q *QueueWaiter) extractPosition(msg string) int {
msg = stripFormatting(msg)
	for _, re := range queuePatterns {
		matches := re.FindStringSubmatch(msg)
		if len(matches) >= 2 {
			n, err := strconv.Atoi(matches[1])
			if err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

func (q *QueueWaiter) extractTotal(msg string) int {
	for _, re := range totalPatterns {
		matches := re.FindStringSubmatch(msg)
		if len(matches) >= 3 {
			n, err := strconv.Atoi(matches[2])
			if err == nil && n > 0 {
				return n
			}
		} else if len(matches) >= 2 {
			n, err := strconv.Atoi(matches[1])
			if err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}
