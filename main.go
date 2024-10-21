package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	channelID := os.Getenv("CHANNEL_ID")
	if botToken == "" || channelID == "" {
		log.Fatal("Error: SLACK_BOT_TOKEN and CHANNEL_ID must be set in the .env file")
	}

	// Parse command-line arguments
	var filePaths string
	flag.StringVar(&filePaths, "files", "", "Comma-separated list of file paths to upload")
	flag.Parse()

	if filePaths == "" {
		log.Fatal("Error: Please provide at least one file path using the -files flag")
	}

	// Create a new Slack API client
	api := slack.New(botToken)

	// Split the file paths and upload each file
	for _, filePath := range strings.Split(filePaths, ",") {
		filePath = strings.TrimSpace(filePath)
		if err := uploadFile(api, channelID, filePath); err != nil {
			log.Printf("Error uploading file %s: %v", filePath, err)
		}
	}
}

func uploadFile(api *slack.Client, channelID, filePath string) error {
	// Check if the file exists and is accessible
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	if fileInfo.Size() == 0 {
		return fmt.Errorf("file is empty")
	}

	// Get the absolute path of the file
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	// Open the file
	file, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Detect file type and set appropriate title and comment
	fileExt := strings.ToLower(filepath.Ext(filePath))
	title := "Uploaded File"
	comment := "Here's the uploaded file"
	switch fileExt {
	case ".pdf":
		title = "PDF Document"
		comment = "Here's the PDF document"
	case ".png", ".jpg", ".jpeg", ".gif":
		title = "Image File"
		comment = "Here's the image file"
	}

	// Prepare upload parameters
	params := slack.UploadFileV2Parameters{
		Channel:        channelID,
		Filename:       filepath.Base(filePath),
		Title:          title,
		InitialComment: comment,
		Reader:         file,
		FileSize:       int(fileInfo.Size()),
	}

	log.Printf("Uploading file '%s' to channel '%s'\n", params.Filename, channelID)

	// Implement retry mechanism
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		fileUpload, err := api.UploadFileV2(params)
		if err == nil {
			log.Printf("File upload successful: %+v\n", fileUpload)
			return nil
		}
		log.Printf("Upload attempt %d failed: %v. Retrying...\n", i+1, err)
		time.Sleep(time.Second * 2) // Wait before retrying
	}

	return fmt.Errorf("failed to upload file after %d attempts", maxRetries)
}
