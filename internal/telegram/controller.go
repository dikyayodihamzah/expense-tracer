package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dikyayodihamzah/expense-tracer/internal/expense"
	"github.com/dikyayodihamzah/expense-tracer/internal/model"
	"github.com/dikyayodihamzah/expense-tracer/internal/pkg/utils"
	"github.com/dikyayodihamzah/expense-tracer/internal/sheets"
	"github.com/dikyayodihamzah/expense-tracer/internal/vision"
)

type Controller struct {
	bot    *tgbotapi.BotAPI
	sheets *sheets.Client
	vision vision.Provider

	mu             sync.Mutex
	pendingExpense map[int64]*model.Expense
}

func New(bot *tgbotapi.BotAPI, sc *sheets.Client, vp vision.Provider) *Controller {
	return &Controller{
		bot:            bot,
		sheets:         sc,
		vision:         vp,
		pendingExpense: make(map[int64]*model.Expense),
	}
}

func (tc *Controller) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
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

func (tc *Controller) handlePhoto(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	tc.reply(chatID, "⏳ Analyzing image...")

	photos := msg.Photo
	largest := photos[len(photos)-1]

	fileURL, err := tc.bot.GetFileDirectURL(largest.FileID)
	if err != nil {
		log.Printf("[handlePhoto] chatID=%d: get file URL: %v", chatID, err)
		tc.reply(chatID, "❌ Failed to get image: "+err.Error())
		return
	}

	imageData, err := utils.DownloadFile(fileURL)
	if err != nil {
		log.Printf("[handlePhoto] chatID=%d: download image: %v", chatID, err)
		tc.reply(chatID, "❌ Failed to download image: "+err.Error())
		return
	}

	e, err := tc.vision.ExtractExpense(ctx, imageData)
	if err != nil {
		log.Printf("[handlePhoto] chatID=%d: extract expense: %v", chatID, err)
		tc.reply(chatID, "❌ Failed to analyze image: "+err.Error())
		return
	}

	if err := expense.FillExpenseMeta(e, time.Now()); err != nil {
		log.Printf("[handlePhoto] chatID=%d: fill expense meta: %v", chatID, err)
		tc.reply(chatID, "❌ Invalid date in parsed expense: "+err.Error())
		return
	}

	tc.mu.Lock()
	tc.pendingExpense[chatID] = e
	tc.mu.Unlock()
	tc.sendConfirmation(chatID, e)
}

func (tc *Controller) handleAdd(ctx context.Context, chatID int64, args string) {
	if strings.TrimSpace(args) == "" {
		tc.reply(chatID, "Usage: /add <description> <nominal> <category>\nExample: /add makan siang 35000 Food")
		return
	}

	e, err := expense.ParseAddCommand(args)
	if err != nil {
		log.Printf("[handleAdd] chatID=%d: parse command: %v", chatID, err)
		tc.reply(chatID, "❌ "+err.Error())
		return
	}

	if err := expense.FillExpenseMeta(e, time.Now()); err != nil {
		log.Printf("[handleAdd] chatID=%d: fill expense meta: %v", chatID, err)
		tc.reply(chatID, "❌ "+err.Error())
		return
	}

	tc.mu.Lock()
	tc.pendingExpense[chatID] = e
	tc.mu.Unlock()
	tc.sendConfirmation(chatID, e)
}

func (tc *Controller) handleDelete(ctx context.Context, chatID int64) {
	if err := tc.sheets.DeleteLastRow(ctx); err != nil {
		log.Printf("[handleDelete] chatID=%d: delete last row: %v", chatID, err)
		tc.reply(chatID, "❌ Failed to delete: "+err.Error())
		return
	}
	tc.reply(chatID, "✅ Last expense deleted.")
}

func (tc *Controller) handleToday(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadTransactions(ctx)
	if err != nil {
		log.Printf("[handleToday] chatID=%d: read transactions: %v", chatID, err)
		tc.reply(chatID, "❌ Failed to read transactions: "+err.Error())
		return
	}
	today := time.Now().Format("02/01/2006")
	tc.replyMarkdown(chatID, expense.FormatToday(rows, today))
}

func (tc *Controller) handleSummary(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadTransactions(ctx)
	if err != nil {
		log.Printf("[handleSummary] chatID=%d: read transactions: %v", chatID, err)
		tc.reply(chatID, "❌ Failed to read transactions: "+err.Error())
		return
	}
	month := time.Now().Format("January")
	tc.replyMarkdown(chatID, expense.FormatSummary(rows, month))
}

func (tc *Controller) handleBudget(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadMonthlyExpense(ctx)
	if err != nil {
		log.Printf("[handleBudget] chatID=%d: read monthly expense: %v", chatID, err)
		tc.reply(chatID, "❌ Failed to read budget: "+err.Error())
		return
	}
	tc.replyMarkdown(chatID, expense.FormatBudget(rows, time.Now()))
}

func (tc *Controller) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	if _, err := tc.bot.Request(tgbotapi.NewCallback(cb.ID, "")); err != nil {
		log.Printf("[handleCallback] chatID=%d: ack callback: %v", chatID, err)
	}

	tc.mu.Lock()
	e, ok := tc.pendingExpense[chatID]
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
		if err := tc.sheets.AppendExpense(ctx, e); err != nil {
			log.Printf("[handleCallback] chatID=%d: append expense: %v", chatID, err)
			tc.reply(chatID, "❌ Failed to save: "+err.Error())
			return
		}
		tc.reply(chatID, fmt.Sprintf("✅ Saved: %s — Rp %s", e.Description, utils.FormatNominal(e.Nominal)))

	case "cancel":
		tc.reply(chatID, "❌ Cancelled.")
	}
}

func (tc *Controller) sendConfirmation(chatID int64, e *model.Expense) {
	text := expense.FormatConfirmation(e)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✓ Save", "confirm"),
			tgbotapi.NewInlineKeyboardButtonData("✗ Cancel", "cancel"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = keyboard
	if _, err := tc.bot.Send(msg); err != nil {
		log.Printf("[sendConfirmation] chatID=%d: send message: %v", chatID, err)
	}
}

func (tc *Controller) reply(chatID int64, text string) {
	if _, err := tc.bot.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		log.Printf("[reply] chatID=%d: send message: %v", chatID, err)
	}
}

func (tc *Controller) replyMarkdown(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := tc.bot.Send(msg); err != nil {
		log.Printf("[replyMarkdown] chatID=%d: send message: %v", chatID, err)
	}
}

// Run starts the Telegram long-polling loop. Blocks until ctx is cancelled.
func (tc *Controller) Run(ctx context.Context) {
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
