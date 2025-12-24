// package main

// import (
// 	"bytes"
// 	"crypto/tls"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"mime/multipart"
// 	"net/http"
// 	"os"
// 	"strings"
// 	"time"

// 	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
// )

// type PDFServiceResponse struct {
// 	Word  string `json:"word"`
// 	Count int    `json:"count"`
// }

// func main() {
// 	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
// 	if botToken == "" {
// 		log.Fatal("TELEGRAM_BOT_TOKEN environment variable not set")
// 	}

// 	// Start health check immediately (critical for Fly.io)
// 	go startHealthServer()

// 	// Initialize bot with retry logic
// 	bot := initializeBotWithRetry(botToken)
// 	if bot == nil {
// 		log.Println("Bot initialization failed, but health server is running")
// 		select {} // Keep app alive
// 	}

// 	log.Printf("Bot authorized as: %s", bot.Self.UserName)

// 	// Check and clean up any existing webhook
// 	cleanupWebhook(bot)

// 	// Start processing updates
// 	processUpdates(bot)
// }

// func startHealthServer() {
// 	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 		w.Write([]byte("OK"))
// 	})

// 	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 		w.Write([]byte("Telegram PDF Bot is running"))
// 	})

// 	port := os.Getenv("PORT")
// 	if port == "" {
// 		port = "8080"
// 	}

// 	log.Printf("Health server started on port %s", port)
// 	log.Fatal(http.ListenAndServe(":"+port, nil))
// }
// func handlePDFDocument(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {

// 	processingMsg := tgbotapi.NewMessage(message.Chat.ID, "⏳ Processing your PDF file...")
// 	sentMsg, _ := bot.Send(processingMsg)

// 	fileURL, err := bot.GetFileDirectURL(message.Document.FileID)
// 	if err != nil {
// 		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to get file: "+err.Error()))
// 		return
// 	}

// 	resp, err := http.Get(fileURL)
// 	if err != nil {
// 		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to download file: "+err.Error()))
// 		return
// 	}
// 	defer resp.Body.Close()

// 	fileContent, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to read file: "+err.Error()))
// 		return
// 	}

// 	body := &bytes.Buffer{}
// 	writer := multipart.NewWriter(body)

// 	part, err := writer.CreateFormFile("file", message.Document.FileName)
// 	if err != nil {
// 		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to prepare file: "+err.Error()))
// 		return
// 	}
// 	part.Write(fileContent)

// 	writer.Close()

// 	req, err := http.NewRequest("POST", "https://statement-parser.fly.dev/count", body)
// 	if err != nil {
// 		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to create request: "+err.Error()))
// 		return
// 	}
// 	req.Header.Set("Content-Type", writer.FormDataContentType())

// 	client := &http.Client{}
// 	serviceResp, err := client.Do(req)
// 	if err != nil {
// 		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to call PDF service: "+err.Error()))
// 		return
// 	}
// 	defer serviceResp.Body.Close()

// 	var result PDFServiceResponse
// 	if err := json.NewDecoder(serviceResp.Body).Decode(&result); err != nil {
// 		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Failed to parse service response: "+err.Error()))
// 		return
// 	}

// 	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, sentMsg.MessageID,
// 		fmt.Sprintf("✅ Analysis complete!\n\n📊 Word: %s\n🔢 Count: %d", result.Word, result.Count))
// 	bot.Send(editMsg)
// }
// func initializeBotWithRetry(token string) *tgbotapi.BotAPI {
// 	maxRetries := 5
// 	retryDelay := 5 * time.Second

// 	for i := 0; i < maxRetries; i++ {
// 		log.Printf("Bot initialization attempt %d/%d", i+1, maxRetries)

// 		// Try with different approaches
// 		var bot *tgbotapi.BotAPI
// 		var err error

// 		// Attempt 1: Standard initialization
// 		bot, err = tgbotapi.NewBotAPI(token)
// 		if err == nil {
// 			return bot
// 		}

// 		log.Printf("Attempt %d failed: %v", i+1, err)

// 		// Attempt 2: With custom client (bypass cert verification)
// 		if i == 1 {
// 			log.Println("Trying with custom HTTP client...")
// 			httpClient := &http.Client{
// 				Timeout: 30 * time.Second,
// 				Transport: &http.Transport{
// 					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
// 				},
// 			}

// 			bot, err = tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, httpClient)
// 			if err == nil {
// 				return bot
// 			}
// 			log.Printf("Custom client also failed: %v", err)
// 		}

// 		if i < maxRetries-1 {
// 			log.Printf("Retrying in %v...", retryDelay)
// 			time.Sleep(retryDelay)
// 		}
// 	}

// 	log.Println("All bot initialization attempts failed")
// 	return nil
// }

// func cleanupWebhook(bot *tgbotapi.BotAPI) {
// 	// Try to delete any existing webhook
// 	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/deleteWebhook", bot.Token)

// 	// First attempt with library
// 	_, err := bot.Request(tgbotapi.DeleteWebhookConfig{})
// 	if err != nil {
// 		log.Printf("DeleteWebhook via library failed: %v", err)
// 	} else {
// 		log.Println("Webhook deleted successfully")
// 		return
// 	}

// 	// Fallback: Direct HTTP call
// 	resp, err := http.Post(apiURL, "application/json", nil)
// 	if err != nil {
// 		log.Printf("Direct deleteWebhook failed: %v", err)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	body, _ := io.ReadAll(resp.Body)
// 	log.Printf("deleteWebhook response: %s", string(body))
// }

// func processUpdates(bot *tgbotapi.BotAPI) {
// 	u := tgbotapi.NewUpdate(0)
// 	u.Timeout = 60
// 	u.Limit = 100
// 	u.Offset = 0

// 	// Get initial updates to clear offset
// 	updates, err := bot.GetUpdates(u)
// 	if err != nil {
// 		log.Printf("Initial GetUpdates failed: %v", err)
// 		log.Println("Will retry in main loop...")
// 	} else if len(updates) > 0 {
// 		log.Printf("Cleared %d pending updates", len(updates))
// 		if len(updates) > 0 {
// 			u.Offset = updates[len(updates)-1].UpdateID + 1
// 		}
// 	}

// 	// Main update loop
// 	for {
// 		updates, err := bot.GetUpdates(u)
// 		if err != nil {
// 			log.Printf("Failed to get updates: %v", err)

// 			// Check if it's a webhook conflict error
// 			if err.Error() == "Conflict: can't use getUpdates method while webhook is active; use deleteWebhook to delete the webhook first" {
// 				log.Println("Webhook conflict detected, attempting to delete webhook...")
// 				cleanupWebhook(bot)
// 				time.Sleep(3 * time.Second)
// 				continue
// 			}

// 			time.Sleep(3 * time.Second)
// 			continue
// 		}

// 		for _, update := range updates {
// 			handleUpdate(&update, bot)
// 			u.Offset = update.UpdateID + 1
// 		}

// 		// Small delay to prevent busy looping
// 		time.Sleep(100 * time.Millisecond)
// 	}
// }

// func handleUpdate(update *tgbotapi.Update, bot *tgbotapi.BotAPI) {
// 	if update.Message == nil {
// 		return
// 	}

// 	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

// 	responseText := "📎 Send me a PDF file and I'll analyze it using the Fly.io service."

// 	if update.Message.Document != nil {
// 		// Check if it's a PDF
// 		isPDF := strings.HasSuffix(strings.ToLower(update.Message.Document.FileName), ".pdf") ||
// 			update.Message.Document.MimeType == "application/pdf"

// 		if isPDF {
// 			responseText = "📄 PDF received! I'll process it with the statement parser service."
// 			// Here you would call your PDF service
// 			go handlePDFDocument(bot, update.Message)
// 		} else {
// 			responseText = "❌ Please send a PDF file (.pdf extension)."
// 		}
// 	}

// 	msg := tgbotapi.NewMessage(update.Message.Chat.ID, responseText)
// 	msg.ReplyToMessageID = update.Message.MessageID

//		if _, err := bot.Send(msg); err != nil {
//			log.Printf("Failed to send message: %v", err)
//		}
//	}
package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}
	}
}
