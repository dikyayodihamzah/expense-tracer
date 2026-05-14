package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"github.com/dikyayodihamzah/expense-tracer/internal/pkg/utils"
	"github.com/dikyayodihamzah/expense-tracer/internal/sheets"
	"github.com/dikyayodihamzah/expense-tracer/internal/telegram"
	"github.com/dikyayodihamzah/expense-tracer/internal/vision"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	token := utils.MustEnv("TELEGRAM_BOT_TOKEN")
	visionProvider := os.Getenv("VISION_PROVIDER")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	googleCreds := utils.MustEnv("GOOGLE_APPLICATION_CREDENTIALS")
	spreadsheetID := utils.MustEnv("SPREADSHEET_ID")
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

	sheetsClient, err := sheets.New(ctx, googleCreds, spreadsheetID, sheetName)
	if err != nil {
		log.Fatalf("failed to create sheets client: %v", err)
	}

	vp, err := vision.New(visionProvider, anthropicKey, openaiKey, googleCreds)
	if err != nil {
		log.Fatalf("failed to create vision provider: %v", err)
	}

	ctrl := telegram.New(bot, sheetsClient, vp)
	ctrl.Run(ctx)

	log.Println("bot stopped")
}
