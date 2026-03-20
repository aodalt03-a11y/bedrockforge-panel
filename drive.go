package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const driveFolderID = "152KOyMR0jRmDRVIYO-ttQyqzFwkQYRX4"

var driveService *drive.Service

func initDrive() error {
	credsJSON := os.Getenv("GOOGLE_CREDS")
	if credsJSON == "" {
		return nil
	}
	ctx := context.Background()
	svc, err := drive.NewService(ctx, option.WithCredentialsJSON([]byte(credsJSON)))
	if err != nil {
		return err
	}
	driveService = svc
	addLog("Google Drive connected")
	return nil
}

func driveReadToken(botName string) ([]byte, error) {
	if driveService == nil { return nil, nil }
	fileName := "token_" + botName + ".json"
	r, err := driveService.Files.List().
		Q("name = '" + fileName + "' and '" + driveFolderID + "' in parents and trashed = false").
		Fields("files(id)").Do()
	if err != nil || len(r.Files) == 0 { return nil, err }
	res, err := driveService.Files.Get(r.Files[0].Id).Download()
	if err != nil { return nil, err }
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func driveWriteToken(botName string, data []byte) error {
	if driveService == nil { return nil }
	fileName := "token_" + botName + ".json"
	r, err := driveService.Files.List().
		Q("name = '" + fileName + "' and '" + driveFolderID + "' in parents and trashed = false").
		Fields("files(id)").Do()
	if err != nil { return err }
	if len(r.Files) > 0 {
		_, err = driveService.Files.Update(r.Files[0].Id, nil).Media(bytes.NewReader(data)).Do()
	} else {
		f := &drive.File{Name: fileName, Parents: []string{driveFolderID}}
		_, err = driveService.Files.Create(f).Media(bytes.NewReader(data)).Do()
	}
	return err
}

func driveListTokens() []string {
	if driveService == nil { return nil }
	r, err := driveService.Files.List().
		Q("'" + driveFolderID + "' in parents and trashed = false").
		Fields("files(name)").Do()
	if err != nil { return nil }
	var names []string
	for _, f := range r.Files {
		if len(f.Name) > 10 {
			names = append(names, f.Name[6:len(f.Name)-5])
		}
	}
	return names
}

func driveTokenToFile(botName, localPath string) error {
	data, err := driveReadToken(botName)
	if err != nil || data == nil { return err }
	return os.WriteFile(localPath, data, 0600)
}

func driveTokenFromFile(botName, localPath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil { return err }
	return driveWriteToken(botName, data)
}

func driveExportCreds() string {
	b, _ := json.MarshalIndent(map[string]string{"status": "connected", "folder": driveFolderID}, "", "  ")
	return string(b)
}
