package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

type UserState string

const (
	StateWelcome   UserState = "welcome"
	StateQuestion  UserState = "question"
	StateCVReview  UserState = "cv_review"
	StateWaitingCV UserState = "waiting_cv"
)

type Bot struct {
	api           *tgbotapi.BotAPI
	adminID       int64
	userSessions  map[int64]*UserSession
	adminMessages map[int]*UserSession
	userStates    map[int64]UserState
}

type UserSession struct {
	UserID       int64
	Username     string
	LastQuestion string
	MessageID    int
	AdminMsgID   int
	HasFile      bool
	FileName     string
	State        UserState
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	adminIDStr := os.Getenv("ADMIN_ID")
	if adminIDStr == "" {
		log.Fatal("ADMIN_ID environment variable is required")
	}

	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		log.Fatal("Invalid ADMIN_ID:", err)
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	faqBot := &Bot{
		api:           bot,
		adminID:       adminID,
		userSessions:  make(map[int64]*UserSession),
		adminMessages: make(map[int]*UserSession),
		userStates:    make(map[int64]UserState),
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			faqBot.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			faqBot.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	userID := message.From.ID
	username := message.From.UserName

	if userID == b.adminID {
		b.handleAdminMessage(message)
	} else {
		b.handleUserQuestion(message, userID, username)
	}
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID

	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	b.api.Request(callbackConfig)

	switch callback.Data {
	case "question":
		b.startQuestionFlow(userID)
	case "cv_review":
		b.startCVReviewFlow(userID)
	}
}

func (b *Bot) handleUserQuestion(message *tgbotapi.Message, userID int64, username string) {
	text := message.Text

	if text == "/start" || text == "/menu" {
		b.showWelcomeMenu(userID)
		return
	}

	currentState, exists := b.userStates[userID]
	if !exists {
		b.showWelcomeMenu(userID)
		return
	}

	switch currentState {
	case StateWelcome:
		b.handleWelcomeState(message, userID, username)
	case StateQuestion:
		b.handleQuestionState(message, userID, username)
	case StateCVReview:
		b.handleCVReviewState(message, userID, username)
	case StateWaitingCV:
		b.handleWaitingCVState(message, userID, username)
	default:
		b.showWelcomeMenu(userID)
	}
}

func (b *Bot) showWelcomeMenu(userID int64) {
	welcomeText := `👋 Welcome! How can I help you today?

Please choose an option:

1️⃣ Ask a Question
2️⃣ CV Review

Type "1" for questions or "2" for CV review.`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Ask Question", "question"),
			tgbotapi.NewInlineKeyboardButtonData("📄 CV Review", "cv_review"),
		),
	)

	msg := tgbotapi.NewMessage(userID, welcomeText)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
	b.userStates[userID] = StateWelcome
}

func (b *Bot) handleWelcomeState(message *tgbotapi.Message, userID int64, username string) {
	text := strings.ToLower(strings.TrimSpace(message.Text))

	if text == "/start" || text == "/menu" || text == "menu" {
		b.showWelcomeMenu(userID)
		return
	}

	if text == "1" || strings.Contains(text, "question") {
		b.startQuestionFlow(userID)
	} else if text == "2" || strings.Contains(text, "cv") || strings.Contains(text, "review") {
		b.startCVReviewFlow(userID)
	} else {
		b.showWelcomeMenu(userID)
	}
}

func (b *Bot) startQuestionFlow(userID int64) {
	instructionText := `❓ Great! I'm here to help answer your questions.

📝 For the best response, please:
• Be specific and clear in your question
• Provide context if needed
• Ask one question at a time

Go ahead and ask your question!`

	msg := tgbotapi.NewMessage(userID, instructionText)
	b.api.Send(msg)
	b.userStates[userID] = StateQuestion
}

func (b *Bot) startCVReviewFlow(userID int64) {
	instructionText := `📄 I'd be happy to review your CV!

📋 To provide the best feedback, please:

1️⃣ Upload your CV to Google Drive
2️⃣ Set sharing permissions to "Anyone with the link can comment"
3️⃣ Copy the Google Drive link
4️⃣ Send me the link here

This allows me to:
✅ Add specific comments to your document
✅ Suggest improvements directly on the text
✅ Track changes and revisions
✅ Provide detailed, actionable feedback

Please share your Google Drive link now:`

	msg := tgbotapi.NewMessage(userID, instructionText)
	b.api.Send(msg)
	b.userStates[userID] = StateCVReview
}

func (b *Bot) handleQuestionState(message *tgbotapi.Message, userID int64, username string) {
	var questionText string
	var hasFile bool
	var fileName string

	if message.Document != nil {
		hasFile = true
		fileName = message.Document.FileName
		questionText = fmt.Sprintf("[File: %s]", fileName)
		if message.Caption != "" {
			questionText = fmt.Sprintf("[File: %s] %s", fileName, message.Caption)
		}
	} else {
		questionText = message.Text
	}

	b.createUserSession(userID, username, questionText, message.MessageID, hasFile, fileName, StateQuestion)
}

func (b *Bot) handleCVReviewState(message *tgbotapi.Message, userID int64, username string) {
	text := message.Text

	if strings.Contains(text, "drive.google.com") || strings.Contains(text, "docs.google.com") {
		questionText := fmt.Sprintf("CV Review Request - Google Drive Link: %s", text)
		b.createUserSession(userID, username, questionText, message.MessageID, false, "", StateCVReview)
	} else if message.Document != nil {
		// fileName := message.Document.FileName
		helpText := `📄 I see you've uploaded a file directly. 

For better collaboration, please upload your CV to Google Drive instead and share the link. This allows me to add comments directly to your document.

Would you like to:
1️⃣ Upload to Google Drive and share the link (recommended)
2️⃣ Continue with the uploaded file

Type "1" for Google Drive or "2" to continue.`

		msg := tgbotapi.NewMessage(userID, helpText)
		b.api.Send(msg)
		b.userStates[userID] = StateWaitingCV
	} else {
		retryText := `❌ Please share a Google Drive link to your CV.

The link should look like:
https://drive.google.com/file/d/your-file-id/view

Or upload your CV to Google Drive first and then share the link here.`

		msg := tgbotapi.NewMessage(userID, retryText)
		b.api.Send(msg)
	}
}

func (b *Bot) handleWaitingCVState(message *tgbotapi.Message, userID int64, username string) {
	text := strings.ToLower(strings.TrimSpace(message.Text))

	if text == "1" {
		b.startCVReviewFlow(userID)
	} else if text == "2" {
		questionText := fmt.Sprintf("CV Review Request - File uploaded directly")
		if message.Document != nil {
			questionText = fmt.Sprintf("CV Review Request - File: %s", message.Document.FileName)
		}
		b.createUserSession(userID, username, questionText, message.MessageID, true, "", StateCVReview)
	} else {
		helpText := `Please choose:
1️⃣ Upload to Google Drive (recommended)
2️⃣ Continue with uploaded file

Type "1" or "2"`

		msg := tgbotapi.NewMessage(userID, helpText)
		b.api.Send(msg)
	}
}

func (b *Bot) createUserSession(userID int64, username, questionText string, messageID int, hasFile bool, fileName string, state UserState) {
	session := &UserSession{
		UserID:       userID,
		Username:     username,
		LastQuestion: questionText,
		MessageID:    messageID,
		HasFile:      hasFile,
		FileName:     fileName,
		State:        state,
	}

	var confirmMsg tgbotapi.MessageConfig
	if state == StateCVReview {
		confirmMsg = tgbotapi.NewMessage(userID, "✅ Thank you for your CV review request! An admin will review it and get back to you with detailed feedback.")
	} else if hasFile {
		confirmMsg = tgbotapi.NewMessage(userID, "✅ Thank you for your question and file! An admin will respond to you shortly.")
	} else {
		confirmMsg = tgbotapi.NewMessage(userID, "✅ Thank you for your question! An admin will respond to you shortly.")
	}
	b.api.Send(confirmMsg)

	var adminNotification string
	var icon string

	switch state {
	case StateCVReview:
		icon = "📄 "
	case StateQuestion:
		if hasFile {
			icon = "📎 "
		} else {
			icon = "❓ "
		}
	default:
		icon = "💬 "
	}

	if username != "" {
		adminNotification = fmt.Sprintf("%sNew message from @%s (ID: %d):\n\n%s\n\n💡 Simply reply to this message to answer the user",
			icon, username, userID, questionText)
	} else {
		adminNotification = fmt.Sprintf("%sNew message from user (ID: %d):\n\n%s\n\n💡 Simply reply to this message to answer the user",
			icon, userID, questionText)
	}

	adminMsg := tgbotapi.NewMessage(b.adminID, adminNotification)
	sent, err := b.api.Send(adminMsg)
	if err == nil {
		session.AdminMsgID = sent.MessageID
		b.userSessions[userID] = session
		b.adminMessages[sent.MessageID] = session
	}

	b.userStates[userID] = StateWelcome
}

func (b *Bot) handleAdminMessage(message *tgbotapi.Message) {
	text := message.Text

	if message.ReplyToMessage != nil {
		session, exists := b.adminMessages[message.ReplyToMessage.MessageID]
		if exists {
			answer := text
			userID := session.UserID

			responseToUser := fmt.Sprintf("Answer to your question:\n\n%s", answer)
			userMsg := tgbotapi.NewMessage(userID, responseToUser)
			_, err := b.api.Send(userMsg)

			if err != nil {
				errorMsg := tgbotapi.NewMessage(b.adminID, fmt.Sprintf("Failed to send message to user: %v", err))
				b.api.Send(errorMsg)
				return
			}

			var confirmationMsg string
			if session.Username != "" {
				confirmationMsg = fmt.Sprintf("✅ Reply sent successfully to @%s", session.Username)
			} else {
				confirmationMsg = fmt.Sprintf("✅ Reply sent successfully to user ID: %d", userID)
			}

			confirmMsg := tgbotapi.NewMessage(b.adminID, confirmationMsg)
			b.api.Send(confirmMsg)

			delete(b.userSessions, userID)
			delete(b.adminMessages, message.ReplyToMessage.MessageID)
			return
		}
	}

	if text == "/sessions" {
		if len(b.userSessions) == 0 {
			msg := tgbotapi.NewMessage(b.adminID, "No active user sessions")
			b.api.Send(msg)
			return
		}

		var sessionsText strings.Builder
		sessionsText.WriteString("Active user sessions:\n\n")

		for _, session := range b.userSessions {
			if session.Username != "" {
				sessionsText.WriteString(fmt.Sprintf("@%s (ID: %d): %s\n\n",
					session.Username, session.UserID, session.LastQuestion))
			} else {
				sessionsText.WriteString(fmt.Sprintf("User ID %d: %s\n\n",
					session.UserID, session.LastQuestion))
			}
		}

		msg := tgbotapi.NewMessage(b.adminID, sessionsText.String())
		b.api.Send(msg)
	} else if text == "/help" {
		helpText := `Admin Commands:
💬 Reply to any question message to answer the user
/sessions - View all active user sessions
/help - Show this help message`

		msg := tgbotapi.NewMessage(b.adminID, helpText)
		b.api.Send(msg)
	}
}

func (b *Bot) isCV(fileName string) bool {
	fileName = strings.ToLower(fileName)

	cvKeywords := []string{"cv", "resume", "curriculum", "vitae"}
	for _, keyword := range cvKeywords {
		if strings.Contains(fileName, keyword) {
			return true
		}
	}

	cvExtensions := []string{".pdf", ".doc", ".docx"}
	for _, ext := range cvExtensions {
		if strings.HasSuffix(fileName, ext) {
			for _, keyword := range cvKeywords {
				if strings.Contains(fileName, keyword) {
					return true
				}
			}
		}
	}

	return false
}
