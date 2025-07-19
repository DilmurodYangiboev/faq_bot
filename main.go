package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
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
	logger        *logrus.Logger
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

func setupLogger() *logrus.Logger {
	logger := logrus.New()

	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	logLevel := os.Getenv("LOG_LEVEL")
	switch strings.ToLower(logLevel) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn", "warning":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.ErrorLevel)
	}

	return logger
}

func main() {
	logger := setupLogger()

	err := godotenv.Load()
	if err != nil {
		logger.Error("No .env file found, using system environment variables")
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		logger.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	adminIDStr := os.Getenv("ADMIN_ID")
	if adminIDStr == "" {
		logger.Fatal("ADMIN_ID environment variable is required")
	}

	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		logger.WithError(err).Fatal("Invalid ADMIN_ID format")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create bot API instance")
	}

	bot.Debug = false

	faqBot := &Bot{
		api:           bot,
		adminID:       adminID,
		userSessions:  make(map[int64]*UserSession),
		adminMessages: make(map[int]*UserSession),
		userStates:    make(map[int64]UserState),
		logger:        logger,
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

	// Log all user entries
	if userID != b.adminID {
		b.logger.WithFields(logrus.Fields{
			"user_id":      userID,
			"username":     username,
			"message_text": message.Text,
			"has_document": message.Document != nil,
		}).Error("USER_ENTRY")
	}

	if userID == b.adminID {
		b.handleAdminMessage(message)
	} else {
		b.handleUserQuestion(message, userID, username)
	}
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID

	// Log user callback interaction
	b.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"username":      callback.From.UserName,
		"callback_data": callback.Data,
	}).Error("USER_CALLBACK")

	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	_, err := b.api.Request(callbackConfig)
	if err != nil {
		b.logger.WithError(err).Error("Failed to answer callback query")
	}

	switch callback.Data {
	case "question":
		b.startQuestionFlow(userID)
	case "cv_review":
		b.startCVReviewFlow(userID)
	case "help":
		b.showUserHelp(userID)
	case "commands":
		b.showUserCommands(userID)
	case "back_to_menu":
		b.showWelcomeMenu(userID)
	case "cancel":
		b.cancelCurrentAction(userID)
	case "1":
		// Handle Google Drive choice for CV upload
		if b.userStates[userID] == StateWaitingCV {
			b.startCVReviewFlow(userID)
		}
	case "2":
		// Handle direct file upload choice for CV
		if b.userStates[userID] == StateWaitingCV {
			// Create a session with direct upload
			b.createUserSession(userID, callback.From.UserName, "CV Review Request - File uploaded directly", 0, true, "", StateCVReview)
		}
	default:
		b.logger.WithFields(logrus.Fields{
			"user_id":       userID,
			"callback_data": callback.Data,
		}).Error("Unknown callback data received")
	}
}

func (b *Bot) handleUserCommands(message *tgbotapi.Message, userID int64) bool {
	text := strings.ToLower(strings.TrimSpace(message.Text))

	switch text {
	case "/start", "/menu", "menu", "main menu", "back":
		b.showWelcomeMenu(userID)
		return true

	case "/question", "/ask", "question", "ask", "ask question":
		b.startQuestionFlow(userID)
		return true

	case "/cv", "/resume", "/cvreview", "cv", "resume", "cv review":
		b.startCVReviewFlow(userID)
		return true

	case "/help", "help":
		b.showUserHelp(userID)
		return true

	case "/commands", "commands":
		b.showUserCommands(userID)
		return true

	case "/cancel", "cancel", "stop":
		b.cancelCurrentAction(userID)
		return true
	}

	return false
}

func (b *Bot) showUserHelp(userID int64) {
	helpText := `🤖 FAQ Bot Help

This bot helps you get answers to your questions and get CV reviews from our admin team.

📝 **How to ask questions:**
• Use /question or just type "question"
• Be specific and clear in your question
• You can attach files if needed

📄 **How to get CV review:**
• Use /cv or just type "cv review"  
• Upload to Google Drive and share the link (recommended)
• Or upload your CV file directly

⚡ **Quick Commands:**
• /start - Main menu
• /question - Ask a question
• /cv - CV review
• /cancel - Cancel current action
• /commands - Show all commands

💡 **Tips:**
• You can type commands or use the buttons
• Type "menu" or "back" to return to main menu anytime
• Type "cancel" to stop current action`

	msg := tgbotapi.NewMessage(userID, helpText)
	_, err := b.api.Send(msg)
	if err != nil {
		b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send user help")
	}
}

func (b *Bot) showUserCommands(userID int64) {
	commandText := `📋 Available Commands:

🏠 **Navigation:**
• /start, /menu - Main menu
• /cancel - Cancel current action

❓ **Questions:**  
• /question, /ask - Ask a question
• question, ask - Same as above

📄 **CV Review:**
• /cv, /resume - CV review
• cv, cv review - Same as above

ℹ️ **Help:**
• /help - Show detailed help
• /commands - Show this list

💡 You can type these commands or just use the buttons!`

	msg := tgbotapi.NewMessage(userID, commandText)
	_, err := b.api.Send(msg)
	if err != nil {
		b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send user commands")
	}
}

func (b *Bot) cancelCurrentAction(userID int64) {
	b.userStates[userID] = StateWelcome

	cancelText := `❌ Action cancelled.

You can start over anytime by:
• Typing /start or /menu
• Using the buttons below
• Typing "question" or "cv review"`

	// Show welcome menu with buttons
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Ask Question", "question"),
			tgbotapi.NewInlineKeyboardButtonData("📄 CV Review", "cv_review"),
		),
	)

	msg := tgbotapi.NewMessage(userID, cancelText)
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	if err != nil {
		b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send cancel message")
	}
}

func (b *Bot) handleUserQuestion(message *tgbotapi.Message, userID int64, username string) {

	// Handle commands first
	if b.handleUserCommands(message, userID) {
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

🎯 **Choose what you need:**

1️⃣ **Ask a Question** - Get answers from our team
2️⃣ **CV Review** - Get professional feedback on your CV

💡 **Quick ways to get started:**
• Click the buttons below
• Type: question, cv review, help
• Use commands: /question, /cv, /help

Need help? Type /help or /commands`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Ask Question", "question"),
			tgbotapi.NewInlineKeyboardButtonData("📄 CV Review", "cv_review"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Help", "help"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Commands", "commands"),
		),
	)

	msg := tgbotapi.NewMessage(userID, welcomeText)
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	if err != nil {
		b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send welcome menu")
		return
	}

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

📝 **For the best response, please:**
• Be specific and clear in your question
• Provide context if needed
• Ask one question at a time
• You can attach files if helpful

💡 **Ready to ask?** Just type your question below!

🔙 **Need to go back?** Type /cancel or /menu`

	// Add a cancel button for easier navigation
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "back_to_menu"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "cancel"),
		),
	)

	msg := tgbotapi.NewMessage(userID, instructionText)
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	if err != nil {
		b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send question flow instructions")
		return
	}

	b.userStates[userID] = StateQuestion
}

func (b *Bot) startCVReviewFlow(userID int64) {
	instructionText := `📄 I'd be happy to review your CV!

📋 **To provide the best feedback, please:**

1️⃣ Upload your CV to Google Drive
2️⃣ Set sharing permissions to "Anyone with the link can comment"
3️⃣ Copy the Google Drive link
4️⃣ Send me the link here

**This allows me to:**
✅ Add specific comments to your document
✅ Suggest improvements directly on the text
✅ Track changes and revisions
✅ Provide detailed, actionable feedback

💡 **Ready?** Share your Google Drive link below!
📎 **Alternative:** You can also upload your CV file directly

🔙 **Need to go back?** Type /cancel or /menu`

	// Add navigation buttons
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "back_to_menu"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "cancel"),
		),
	)

	msg := tgbotapi.NewMessage(userID, instructionText)
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	if err != nil {
		b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send CV review flow instructions")
		return
	}

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
		helpText := `📄 I see you've uploaded a file directly. 

For better collaboration, please upload your CV to Google Drive instead and share the link. This allows me to add comments directly to your document.

Would you like to:
1️⃣ Upload to Google Drive and share the link (recommended)
2️⃣ Continue with the uploaded file

Type "1" for Google Drive or "2" to continue.`

		msg := tgbotapi.NewMessage(userID, helpText)
		_, err := b.api.Send(msg)
		if err != nil {
			b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send file upload help message")
			return
		}

		b.userStates[userID] = StateWaitingCV
	} else {
		retryText := `❌ Please share a Google Drive link to your CV.

The link should look like:
https://drive.google.com/file/d/your-file-id/view

Or upload your CV to Google Drive first and then share the link here.`

		msg := tgbotapi.NewMessage(userID, retryText)
		_, err := b.api.Send(msg)
		if err != nil {
			b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send CV retry message")
		}
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

1️⃣ **Upload to Google Drive** (recommended)
2️⃣ **Continue with uploaded file**

Type "1" or "2", or use the commands below:

🔙 **Back to menu:** /menu or /cancel`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📁 Google Drive", "1"),
				tgbotapi.NewInlineKeyboardButtonData("📎 Upload File", "2"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Menu", "back_to_menu"),
			),
		)

		msg := tgbotapi.NewMessage(userID, helpText)
		msg.ReplyMarkup = keyboard
		_, err := b.api.Send(msg)
		if err != nil {
			b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send CV choice help message")
		}
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

	_, err := b.api.Send(confirmMsg)
	if err != nil {
		b.logger.WithError(err).WithField("user_id", userID).Error("Failed to send confirmation message to user")
		return
	}

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
	if err != nil {
		b.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  userID,
			"admin_id": b.adminID,
		}).Error("Failed to send notification to admin")
		return
	}

	session.AdminMsgID = sent.MessageID
	b.userSessions[userID] = session
	b.adminMessages[sent.MessageID] = session

	b.userStates[userID] = StateWelcome
}

func (b *Bot) handleAdminMessage(message *tgbotapi.Message) {
	text := message.Text

	if message.ReplyToMessage != nil {
		replyToMsgID := message.ReplyToMessage.MessageID
		session, exists := b.adminMessages[replyToMsgID]
		if exists {
			answer := text
			userID := session.UserID

			responseToUser := fmt.Sprintf("Answer to your question:\n\n%s", answer)
			userMsg := tgbotapi.NewMessage(userID, responseToUser)
			_, err := b.api.Send(userMsg)

			if err != nil {
				b.logger.WithError(err).WithFields(logrus.Fields{
					"user_id":  userID,
					"admin_id": b.adminID,
				}).Error("Failed to send admin reply to user")
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
			_, err = b.api.Send(confirmMsg)
			if err != nil {
				b.logger.WithError(err).Error("Failed to send confirmation to admin")
			}

			delete(b.userSessions, userID)
			delete(b.adminMessages, replyToMsgID)
			return
		}
	}

	if text == "/sessions" {
		if len(b.userSessions) == 0 {
			msg := tgbotapi.NewMessage(b.adminID, "No active user sessions")
			_, err := b.api.Send(msg)
			if err != nil {
				b.logger.WithError(err).Error("Failed to send 'no sessions' message")
			}
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
		_, err := b.api.Send(msg)
		if err != nil {
			b.logger.WithError(err).Error("Failed to send sessions list")
		}
	} else if text == "/help" {
		helpText := `Admin Commands:
💬 Reply to any question message to answer the user
/sessions - View all active user sessions
/help - Show this help message`

		msg := tgbotapi.NewMessage(b.adminID, helpText)
		_, err := b.api.Send(msg)
		if err != nil {
			b.logger.WithError(err).Error("Failed to send help message")
		}
	}
}
