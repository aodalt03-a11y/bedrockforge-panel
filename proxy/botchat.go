package proxy

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func SendBotChat(msg string) {
	if activeBot == nil || activeBot.server == nil { return }
	activeBot.server.WritePacket(&packet.Text{
		TextType: packet.TextTypeChat,
		Message:  msg,
	})
}
