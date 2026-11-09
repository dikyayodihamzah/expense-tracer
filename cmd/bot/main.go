package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"github.com/dikyayodihamzah/expense-tracer/internal/gauth"
	"github.com/dikyayodihamzah/expense-tracer/internal/model"
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
	googleCreds := utils.MustEnv("GOOGLE_CREDENTIALS_FILE")
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

	googleClient, googleTS, err := gauth.NewClient(ctx, googleCreds)
	if err != nil {
		log.Fatalf("failed to create google client: %v", err)
	}

	sheetsClient, err := sheets.New(ctx, googleClient, spreadsheetID, sheetName)
	if err != nil {
		log.Fatalf("failed to create sheets client: %v", err)
	}

	vp, err := vision.New(visionProvider, anthropicKey, openaiKey, googleTS)
	if err != nil {
		log.Fatalf("failed to create vision provider: %v", err)
	}

	// The pocket tracker's typed-in "Used" cells; override when the table moves.
	pocketCells := map[model.Account]string{
		model.AccountBRI:   envOr("POCKET_CELL_BRI", "Monthly Expense!J4"),
		model.AccountCash:  envOr("POCKET_CELL_CASH", "Monthly Expense!J5"),
		model.AccountGoPay: envOr("POCKET_CELL_GOPAY", "Monthly Expense!J8"),
	}

	// Rows from the first pocket group down; reading stops at the "Total" row.
	pocketTable := envOr("POCKET_TABLE", "Monthly Expense!F3:K40")

	// Chat that gets the month-end checklist; unset disables the reminder.
	var reminderChatID int64
	if v := os.Getenv("REMINDER_CHAT_ID"); v != "" {
		if reminderChatID, err = strconv.ParseInt(v, 10, 64); err != nil {
			log.Fatalf("invalid REMINDER_CHAT_ID %q: %v", v, err)
		}
	}

	ctrl := telegram.New(bot, sheetsClient, vp, pocketCells, pocketTable, reminderChatID)
	ctrl.Run(ctx)

	log.Println("bot stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
