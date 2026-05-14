package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"github.com/dikyayodihamzah/expense-tracer/internal/controller"
	"github.com/dikyayodihamzah/expense-tracer/internal/repository"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	token := mustEnv("TELEGRAM_BOT_TOKEN")
	visionProvider := os.Getenv("VISION_PROVIDER")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	googleCreds := mustEnv("GOOGLE_APPLICATION_CREDENTIALS")
	spreadsheetID := mustEnv("SPREADSHEET_ID")
	sheetName := os.Getenv("SHEET_NAME")
	if sheetName == "" {
		sheetName = "Transaction 2026"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}
	log.Printf("authorized as @%s", bot.Self.UserName)

	sheetsClient, err := repository.NewSheetsClient(ctx, googleCreds, spreadsheetID, sheetName)
	if err != nil {
		log.Fatalf("failed to create sheets client: %v", err)
	}

	vision, err := repository.NewVisionProvider(visionProvider, anthropicKey, openaiKey, googleCreds)
	if err != nil {
		log.Fatalf("failed to create vision provider: %v", err)
	}

	ctrl := controller.NewTelegramController(bot, sheetsClient, vision)
	ctrl.Run(ctx)

	log.Println("bot stopped")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return v
}
