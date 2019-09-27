package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/nlopes/slack"
	"os"
)

type Configuration struct {
	SlackToken string
	ChannelId  string
}

func loadConfig() Configuration {
	gopath := os.Getenv("GOPATH")
	file, _ := os.Open(gopath + "/src/lax/conf.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)

	if err != nil {
		panic(err)
	}

	defer file.Close()
	return configuration
}

func setFlags() (string, string, string, string, bool) {
	var msg, title, fileType, filePath string
	var inlineFlag bool

	flag.StringVar(&msg, "message", "", "message to accompany snippet in post")
	flag.StringVar(&title, "title", "output", "title for uploaded file/output to be attached to message")
	flag.StringVar(&fileType, "type", "auto", "filetype for upload (used for syntax highlighting)")
	flag.BoolVar(&inlineFlag, "inline", false, "Show message inline. Syntax highlighting will not work with embedded code.")
	flag.StringVar(&filePath, "file", "", "Pass a file to be sent to Slack. Will be ignored if input is piped through STDIN")

	flag.Parse()
	return msg, title, fileType, filePath, inlineFlag
}

func main() {
	// standard input file descriptor
	fileInfo, err := os.Stdin.Stat()

	config := loadConfig()
	SlackToken := config.SlackToken
	ChannelId := config.ChannelId
	api := slack.New(SlackToken)

	msg, title, fileType, filePath, inlineFlag := setFlags()

	if err != nil {
		panic(err)
	}

	var outputBuffer bytes.Buffer

	if fileInfo.Mode()&os.ModeCharDevice == 0 {
		// input is from unix pipe
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			outputBuffer.WriteString(scanner.Text())
			outputBuffer.WriteString("\n")
		}
	} else if filePath != "" {
		// file path passed as arg
		file, err := os.Open(filePath)

		if err != nil {
			fmt.Printf("File Open error: %s", err)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			outputBuffer.WriteString(scanner.Text())
			outputBuffer.WriteString("\n")
		}
	} else {
		//no input
		fmt.Printf("Usage:\n")
		fmt.Printf("  $ <INPUT> | lax OR $ lax -file <FILEPATH>\n")
		return
	}

	if !inlineFlag {
		// upload input as file attachment
		slackUploadParams := slack.FileUploadParameters{
			InitialComment: msg,
			Filetype:       fileType,
			Title:          title,
			Channels:       []string{ChannelId},
			Content:        outputBuffer.String(),
		}

		_, err := api.UploadFile(slackUploadParams)

		if err != nil {
			fmt.Printf("Slack Error: %s\n", err)
			return
		}
		fmt.Println("message sent to slack")

	} else {
		// upload as message attachment
		slackAttachment := slack.Attachment{
			Pretext: msg,
			Text:    "```\n" + outputBuffer.String() + "\n```",
		}

		channelId, _, err := api.PostMessage(ChannelId, slack.MsgOptionAttachments(slackAttachment))
		if err != nil {
			fmt.Println("%s\n", err)
			return
		}

		fmt.Printf("message sent to channel %s\n", channelId)
	}
}
