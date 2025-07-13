# FAQ Telegram Bot

A Telegram bot that allows users to ask questions and admins to reply directly to specific users.

## Features

- Users can send questions to the bot
- Admin receives notifications with user details and questions
- Admin can reply to specific users using commands
- User sessions are tracked until answered
- Admin can view all active sessions

## Setup

1. Create a new bot with [@BotFather](https://t.me/BotFather) on Telegram
2. Get your bot token
3. Get your Telegram user ID (you can use [@userinfobot](https://t.me/userinfobot))
4. Create a `.env` file:
   ```bash
   cp .env.example .env
   ```
   Then edit `.env` with your credentials:
   ```
   TELEGRAM_BOT_TOKEN=your_bot_token_here
   ADMIN_ID=your_telegram_user_id_here
   ```

## Running

```bash
go mod tidy
go run main.go
```

## Admin Commands

- 💬 **Reply to any question message** - Simply use Telegram's reply feature on question notifications
- `/sessions` - View all active user sessions
- `/help` - Show help message

## Usage Flow

1. User sends a question to the bot
2. Bot confirms receipt to user
3. Admin receives notification with user info and question
4. **Admin simply replies to the notification message**
5. User receives the answer
6. Session is closed