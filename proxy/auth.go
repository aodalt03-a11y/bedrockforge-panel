package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/auth"
	"golang.org/x/oauth2"
)

const tokenFile = "token.json"

type urlWriter struct{}

func (w urlWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	if len(s) > 4 && s[:4] == "http" {
		fmt.Print("[AUTH_URL]" + s)
	} else {
		os.Stdout.Write(p)
	}
	return len(p), nil
}

func loadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	var t oauth2.Token
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func saveToken(t *oauth2.Token) {
	data, err := json.Marshal(t)
	if err != nil {
		log.Printf("[BF] failed to marshal token: %v", err)
		return
	}
	if err := os.WriteFile(tokenFile, data, 0600); err != nil {
		log.Printf("[BF] failed to save token: %v", err)
		return
	}
	log.Println("[BF] token saved")
}

func getTokenSource() (oauth2.TokenSource, error) {
	if t, err := loadToken(); err == nil {
		log.Println("[BF] loaded saved Xbox token")
		src := auth.RefreshTokenSource(t)
		go func() {
			time.Sleep(2 * time.Second)
			if newToken, err := src.Token(); err == nil {
				saveToken(newToken)
			}
		}()
		return src, nil
	}
	log.Println("[BF] authenticating with Xbox Live...")
	liveToken, err := auth.RequestLiveTokenWriter(urlWriter{})
	if err != nil {
		return nil, fmt.Errorf("Xbox auth: %w", err)
	}
	saveToken(liveToken)
	return auth.RefreshTokenSource(liveToken), nil
}
