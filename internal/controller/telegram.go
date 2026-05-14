package controller

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/repository"
	"github.com/dikyayodihamzah/expense-tracer/internal/service"
)

type TelegramController struct {
	bot    *tgbotapi.BotAPI
	sheets *repository.SheetsClient
	vision repository.VisionProvider

	mu sync.Mutex
	// pendingExpense holds a parsed expense awaiting user confirmation.
	// Key: chatID, Value: *model.Expense
	pendingExpense map[int64]*model.Expense
}

func NewTelegramController(bot *tgbotapi.BotAPI, sheets *repository.SheetsClient, vision repository.VisionProvider) *TelegramController {
	return &TelegramController{
		bot:            bot,
		sheets:         sheets,
		vision:         vision,
		pendingExpense: make(map[int64]*model.Expense),
	}
}

func (tc *TelegramController) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		tc.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	msg := update.Message
	chatID := msg.Chat.ID

	if msg.Photo != nil {
		tc.handlePhoto(ctx, msg)
		return
	}

	if !msg.IsCommand() {
		return
	}

	switch msg.Command() {
	case "add":
		tc.handleAdd(ctx, chatID, msg.CommandArguments())
	case "delete":
		tc.handleDelete(ctx, chatID)
	case "today":
		tc.handleToday(ctx, chatID)
	case "summary":
		tc.handleSummary(ctx, chatID)
	case "budget":
		tc.handleBudget(ctx, chatID)
	}
}

func (tc *TelegramController) handlePhoto(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	tc.reply(chatID, "⏳ Analyzing image...")

	// Get the largest photo size.
	photos := msg.Photo
	largest := photos[len(photos)-1]

	fileURL, err := tc.bot.GetFileDirectURL(largest.FileID)
	if err != nil {
		tc.reply(chatID, "❌ Failed to get image: "+err.Error())
		return
	}

	imageData, err := downloadFile(fileURL)
	if err != nil {
		tc.reply(chatID, "❌ Failed to download image: "+err.Error())
		return
	}

	expense, err := tc.vision.ExtractExpense(ctx, imageData)
	if err != nil {
		tc.reply(chatID, "❌ Failed to analyze image: "+err.Error())
		return
	}

	if err := service.FillExpenseMeta(expense, time.Now()); err != nil {
		tc.reply(chatID, "❌ Invalid date in parsed expense: "+err.Error())
		return
	}

	tc.mu.Lock()
	tc.pendingExpense[chatID] = expense
	tc.mu.Unlock()
	tc.sendConfirmation(chatID, expense)
}

func (tc *TelegramController) handleAdd(ctx context.Context, chatID int64, args string) {
	if strings.TrimSpace(args) == "" {
		tc.reply(chatID, "Usage: /add <description> <nominal> <category>\nExample: /add makan siang 35000 Food")
		return
	}

	expense, err := service.ParseAddCommand(args)
	if err != nil {
		tc.reply(chatID, "❌ "+err.Error())
		return
	}

	if err := service.FillExpenseMeta(expense, time.Now()); err != nil {
		tc.reply(chatID, "❌ "+err.Error())
		return
	}

	tc.mu.Lock()
	tc.pendingExpense[chatID] = expense
	tc.mu.Unlock()
	tc.sendConfirmation(chatID, expense)
}

func (tc *TelegramController) handleDelete(ctx context.Context, chatID int64) {
	if err := tc.sheets.DeleteLastRow(ctx); err != nil {
		tc.reply(chatID, "❌ Failed to delete: "+err.Error())
		return
	}
	tc.reply(chatID, "✅ Last expense deleted.")
}

func (tc *TelegramController) handleToday(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadTransactions(ctx)
	if err != nil {
		tc.reply(chatID, "❌ Failed to read transactions: "+err.Error())
		return
	}
	today := time.Now().Format("02/01/2006")
	tc.replyMarkdown(chatID, service.FormatToday(rows, today))
}

func (tc *TelegramController) handleSummary(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadTransactions(ctx)
	if err != nil {
		tc.reply(chatID, "❌ Failed to read transactions: "+err.Error())
		return
	}
	month := time.Now().Format("January")
	tc.replyMarkdown(chatID, service.FormatSummary(rows, month))
}

func (tc *TelegramController) handleBudget(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadMonthlyExpense(ctx)
	if err != nil {
		tc.reply(chatID, "❌ Failed to read budget: "+err.Error())
		return
	}
	tc.replyMarkdown(chatID, service.FormatBudget(rows))
}

func (tc *TelegramController) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	tc.bot.Request(tgbotapi.NewCallback(cb.ID, ""))

	tc.mu.Lock()
	expense, ok := tc.pendingExpense[chatID]
	if ok {
		delete(tc.pendingExpense, chatID)
	}
	tc.mu.Unlock()

	if !ok {
		tc.reply(chatID, "No pending expense found. Please submit again.")
		return
	}

	switch cb.Data {
	case "confirm":
		if err := tc.sheets.AppendExpense(ctx, expense); err != nil {
			tc.reply(chatID, "❌ Failed to save: "+err.Error())
			return
		}
		tc.reply(chatID, fmt.Sprintf("✅ Saved: %s — Rp %d", expense.Description, expense.Nominal))

	case "cancel":
		tc.reply(chatID, "❌ Cancelled.")
	}
}

func (tc *TelegramController) sendConfirmation(chatID int64, e *model.Expense) {
	text := service.FormatConfirmation(e)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✓ Save", "confirm"),
			tgbotapi.NewInlineKeyboardButtonData("✗ Cancel", "cancel"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = keyboard
	tc.bot.Send(msg)
}

func (tc *TelegramController) reply(chatID int64, text string) {
	tc.bot.Send(tgbotapi.NewMessage(chatID, text))
}

func (tc *TelegramController) replyMarkdown(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	tc.bot.Send(msg)
}

func downloadFile(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024)) // 20 MB max
}

// Run starts the Telegram long-polling loop. Blocks until ctx is cancelled.
func (tc *TelegramController) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := tc.bot.GetUpdatesChan(u)

	log.Println("bot is running, waiting for updates...")
	for {
		select {
		case <-ctx.Done():
			tc.bot.StopReceivingUpdates()
			return
		case update := <-updates:
			go tc.HandleUpdate(ctx, update)
		}
	}
}
