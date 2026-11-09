package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
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
	// pocketCells maps a tracked account to its "Used" cell in the pocket tracker.
	pocketCells map[model.Account]string
	// pocketTable is the pocket tracker range read by /pockets.
	pocketTable string

	mu             sync.Mutex
	pendingExpense map[int64][]*model.Expense
}

func New(bot *tgbotapi.BotAPI, sc *sheets.Client, vp vision.Provider, pocketCells map[model.Account]string, pocketTable string) *Controller {
	return &Controller{
		bot:            bot,
		sheets:         sc,
		vision:         vp,
		pocketCells:    pocketCells,
		pocketTable:    pocketTable,
		pendingExpense: make(map[int64][]*model.Expense),
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
	case "today":
		tc.handleToday(ctx, chatID)
	case "summary":
		tc.handleSummary(ctx, chatID)
	case "budget":
		tc.handleBudget(ctx, chatID)
	case "pockets":
		tc.handlePockets(ctx, chatID)
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

	es, err := tc.vision.ExtractExpenses(ctx, imageData)
	if err != nil {
		log.Printf("[handlePhoto] chatID=%d: extract expense: %v", chatID, err)
		tc.reply(chatID, "❌ Failed to analyze image: "+err.Error())
		return
	}

	for _, e := range es {
		if err := expense.FillExpenseMeta(e, time.Now()); err != nil {
			log.Printf("[handlePhoto] chatID=%d: fill expense meta: %v", chatID, err)
			tc.reply(chatID, "❌ Invalid date in parsed expense: "+err.Error())
			return
		}
	}

	tc.mu.Lock()
	tc.pendingExpense[chatID] = es
	tc.mu.Unlock()
	tc.showConfirmation(chatID, 0, es)
}

func (tc *Controller) handleAdd(ctx context.Context, chatID int64, args string) {
	if strings.TrimSpace(args) == "" {
		tc.reply(chatID, "Usage: /add <description> <nominal> <category> [@account] [DD/MM/YYYY]\nExample: /add makan siang 35000 Eating Out\nPaid with GoPay: /add ojek 8000 Transportation @gopay\nWith date: /add makan siang 35000 Eating Out 17/10/2026 (no date = today, no account = Jago Main)")
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

	es := []*model.Expense{e}
	tc.mu.Lock()
	tc.pendingExpense[chatID] = es
	tc.mu.Unlock()
	tc.showConfirmation(chatID, 0, es)
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

func (tc *Controller) handlePockets(ctx context.Context, chatID int64) {
	rows, err := tc.sheets.ReadPockets(ctx, tc.pocketTable)
	if err != nil {
		log.Printf("[handlePockets] chatID=%d: read pockets: %v", chatID, err)
		tc.reply(chatID, "❌ Failed to read pockets: "+err.Error())
		return
	}
	tc.replyMarkdown(chatID, expense.FormatPockets(rows))
}

func (tc *Controller) handleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID
	if _, err := tc.bot.Request(tgbotapi.NewCallback(cb.ID, "")); err != nil {
		log.Printf("[handleCallback] chatID=%d: ack callback: %v", chatID, err)
	}

	tc.mu.Lock()
	es, ok := tc.pendingExpense[chatID]
	tc.mu.Unlock()
	if !ok {
		tc.reply(chatID, "No pending expense found. Please submit again.")
		return
	}

	// Callback data is "action" or "action:arg", e.g. "setcat:0:3".
	action, arg, _ := strings.Cut(cb.Data, ":")
	switch action {
	case "confirm":
		tc.closeConfirmation(chatID, msgID)
		tc.save(ctx, chatID, es)

	case "cancel":
		tc.closeConfirmation(chatID, msgID)
		tc.reply(chatID, "❌ Cancelled.")

	case "back":
		tc.showConfirmation(chatID, msgID, es)

	case "editcat":
		if len(es) == 1 {
			tc.edit(chatID, msgID, "🗂 Pick a category:", categoryKeyboard(0))
			return
		}
		tc.edit(chatID, msgID, "🗂 Which expense?", expenseKeyboard(es))

	case "pickcat":
		i, err := strconv.Atoi(arg)
		if err != nil || i < 0 || i >= len(es) {
			tc.reply(chatID, "That expense is no longer pending. Please submit again.")
			return
		}
		tc.edit(chatID, msgID, fmt.Sprintf("🗂 Category for %s:", es[i].Description), categoryKeyboard(i))

	case "setcat":
		iStr, kStr, _ := strings.Cut(arg, ":")
		i, err1 := strconv.Atoi(iStr)
		k, err2 := strconv.Atoi(kStr)
		cats := model.AllCategories()
		if err1 != nil || err2 != nil || i < 0 || i >= len(es) || k < 0 || k >= len(cats) {
			tc.reply(chatID, "That expense is no longer pending. Please submit again.")
			return
		}
		tc.mu.Lock()
		es[i].Category = cats[k]
		tc.mu.Unlock()
		tc.showConfirmation(chatID, msgID, es)

	case "editacc":
		tc.edit(chatID, msgID, "💳 Paid from:", accountKeyboard())

	case "setacc":
		k, err := strconv.Atoi(arg)
		accs := model.AllAccounts()
		if err != nil || k < 0 || k >= len(accs) {
			tc.reply(chatID, "That expense is no longer pending. Please submit again.")
			return
		}
		// One image is paid from one source, so the account applies to every entry.
		tc.mu.Lock()
		for _, e := range es {
			e.Account = accs[k]
		}
		tc.mu.Unlock()
		tc.showConfirmation(chatID, msgID, es)
	}
}

// save appends es to the sheet, then updates the pocket tracker for what was saved.
func (tc *Controller) save(ctx context.Context, chatID int64, es []*model.Expense) {
	tc.mu.Lock()
	delete(tc.pendingExpense, chatID)
	tc.mu.Unlock()

	for i, e := range es {
		if err := tc.sheets.AppendExpense(ctx, e); err != nil {
			log.Printf("[save] chatID=%d: append expense: %v", chatID, err)
			// Keep the unsaved ones so tapping Save again retries only those.
			tc.mu.Lock()
			tc.pendingExpense[chatID] = es[i:]
			tc.mu.Unlock()
			lines := append([]string{fmt.Sprintf("❌ Failed to save %q (%d of %d saved): %v", e.Description, i, len(es), err)},
				tc.applyPockets(ctx, chatID, es[:i])...)
			tc.reply(chatID, strings.Join(lines, "\n"))
			tc.showConfirmation(chatID, 0, es[i:])
			return
		}
	}
	lines := append([]string{expense.FormatSaved(es)}, tc.applyPockets(ctx, chatID, es)...)
	tc.reply(chatID, strings.Join(lines, "\n"))
}

// applyPockets adds saved daily-living spending to the pocket tracker and returns
// the lines to report: tracker updates and the transfers still to do by hand.
func (tc *Controller) applyPockets(ctx context.Context, chatID int64, saved []*model.Expense) []string {
	adds, reminders := expense.PocketActions(saved)
	var lines []string
	for _, acc := range model.AllAccounts() {
		amount, cell := adds[acc], tc.pocketCells[acc]
		if amount == 0 || cell == "" {
			continue
		}
		if err := tc.sheets.AddToCell(ctx, cell, amount); err != nil {
			log.Printf("[applyPockets] chatID=%d: add to %s: %v", chatID, cell, err)
			lines = append(lines, fmt.Sprintf("⚠️ Could not update the %s tracker (%s): add Rp %s by hand", acc, cell, utils.FormatNominal(amount)))
			continue
		}
		lines = append(lines, fmt.Sprintf("📊 %s tracker +Rp %s", acc, utils.FormatNominal(amount)))
	}
	return append(lines, reminders...)
}

// showConfirmation sends the pending expenses with their action buttons, or redraws
// an existing message in place when msgID is set (after an edit).
func (tc *Controller) showConfirmation(chatID int64, msgID int, es []*model.Expense) {
	text := expense.FormatConfirmations(es)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✓ Save", "confirm"),
			tgbotapi.NewInlineKeyboardButtonData("✗ Cancel", "cancel"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗂 Edit category", "editcat"),
			tgbotapi.NewInlineKeyboardButtonData("💳 Edit account", "editacc"),
		),
	)
	if msgID != 0 {
		cfg := tgbotapi.NewEditMessageTextAndMarkup(chatID, msgID, text, keyboard)
		cfg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := tc.bot.Send(cfg); err != nil {
			log.Printf("[showConfirmation] chatID=%d: edit message: %v", chatID, err)
		}
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = keyboard
	if _, err := tc.bot.Send(msg); err != nil {
		log.Printf("[showConfirmation] chatID=%d: send message: %v", chatID, err)
	}
}

// closeConfirmation removes the buttons so a finished confirmation can't be tapped again.
func (tc *Controller) closeConfirmation(chatID int64, msgID int) {
	empty := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
	if _, err := tc.bot.Send(tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, empty)); err != nil {
		log.Printf("[closeConfirmation] chatID=%d: remove buttons: %v", chatID, err)
	}
}

func (tc *Controller) edit(chatID int64, msgID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	if _, err := tc.bot.Send(tgbotapi.NewEditMessageTextAndMarkup(chatID, msgID, text, keyboard)); err != nil {
		log.Printf("[edit] chatID=%d: edit message: %v", chatID, err)
	}
}

func expenseKeyboard(es []*model.Expense) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, e := range es {
		label := fmt.Sprintf("%d. %s — %s", i+1, e.Category, e.Description)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("pickcat:%d", i))))
	}
	return withBack(rows)
}

func categoryKeyboard(i int) tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton
	for k, c := range model.AllCategories() {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(string(c), fmt.Sprintf("setcat:%d:%d", i, k)))
	}
	return withBack(pairs(buttons))
}

func accountKeyboard() tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton
	for k, a := range model.AllAccounts() {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(string(a), fmt.Sprintf("setacc:%d", k)))
	}
	return withBack(pairs(buttons))
}

// pairs lays buttons out two per row.
func pairs(buttons []tgbotapi.InlineKeyboardButton) [][]tgbotapi.InlineKeyboardButton {
	var rows [][]tgbotapi.InlineKeyboardButton
	for len(buttons) > 2 {
		rows = append(rows, buttons[:2])
		buttons = buttons[2:]
	}
	return append(rows, buttons)
}

func withBack(rows [][]tgbotapi.InlineKeyboardButton) tgbotapi.InlineKeyboardMarkup {
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("« Back", "back")))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
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
