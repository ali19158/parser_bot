package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type PDFServiceResponse struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

func main() {

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Document != nil {
			go handlePDFDocument(bot, update.Message)
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"📎 Please send me a PDF file to analyze. I will count occurrences of 'Пополнение'")
			bot.Send(msg)
		}
	}
}

func handlePDFDocument(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {

	processingMsg := tgbotapi.NewMessage(message.Chat.ID, "⏳ Processing your PDF file...")
	sentMsg, _ := bot.Send(processingMsg)

	fileURL, err := bot.GetFileDirectURL(message.Document.FileID)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to get file: "+err.Error()))
		return
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to download file: "+err.Error()))
		return
	}
	defer resp.Body.Close()

	fileContent, err := io.ReadAll(resp.Body)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to read file: "+err.Error()))
		return
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", message.Document.FileName)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to prepare file: "+err.Error()))
		return
	}
	part.Write(fileContent)

	writer.Close()

	req, err := http.NewRequest("POST", "https://statement-parser.fly.dev/count", body)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to create request: "+err.Error()))
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	serviceResp, err := client.Do(req)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to call PDF service: "+err.Error()))
		return
	}
	defer serviceResp.Body.Close()

	var result PDFServiceResponse
	if err := json.NewDecoder(serviceResp.Body).Decode(&result); err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to parse service response: "+err.Error()))
		return
	}

	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentMsg.MessageID,
		fmt.Sprintf("✅ Analysis complete!\n\n📊 Word: %s\n🔢 Count: %d", result.Word, result.Count))
	bot.Send(editMsg)
}
