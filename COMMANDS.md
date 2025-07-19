# FAQ Bot Commands & Features

## 🚀 Quick Start Commands

### Main Navigation
- `/start` - Start the bot and show main menu
- `/menu` - Return to main menu anytime
- `/cancel` - Cancel current action and go back to menu

### Core Features
- `/question` or `/ask` - Ask a question 
- `/cv` or `/resume` - Request CV review

### Help & Information
- `/help` - Show detailed help and instructions
- `/commands` - Show this command list

## 💬 Natural Language Commands

You don't need to use `/` commands! The bot understands natural language:

### Asking Questions
- `question`
- `ask`
- `ask question`
- `I have a question`

### CV Review
- `cv`
- `resume` 
- `cv review`
- `review my cv`

### Navigation
- `menu`
- `main menu`
- `back`
- `cancel`
- `stop`
- `help`

## 🎯 How to Use

### 1. Ask Questions
1. Type `/question` or just `question`
2. Type your question clearly
3. Optionally attach files
4. Wait for admin response

### 2. Get CV Review
1. Type `/cv` or just `cv review`
2. **Recommended:** Upload to Google Drive and share link
3. **Alternative:** Upload CV file directly
4. Wait for detailed feedback

### 3. Navigation Tips
- Use buttons for easy navigation
- Type commands for quick access
- Use `/cancel` to exit any flow
- Type `/menu` to return to main menu anytime

## 🤖 Interactive Buttons

The bot provides interactive buttons for:
- ❓ Ask Question
- 📄 CV Review  
- ℹ️ Help
- 📋 Commands
- 🔙 Back to Menu
- ❌ Cancel

## 💡 Pro Tips

1. **Flexible Commands:** You can type commands with or without `/`
2. **Smart Recognition:** The bot understands variations like "question", "ask", "cv review"
3. **Always Available:** Navigation commands work from any state
4. **File Support:** Attach files to questions for better context
5. **Google Drive:** Use Google Drive links for CV reviews to get direct comments

## 🔧 Admin Commands

### For Bot Administrator
- `/sessions` - View all active user sessions
- `/help` - Show admin help
- **Reply to messages** - Answer user questions directly

## 📝 Example Usage

```
User: /start
Bot: Shows welcome menu with buttons

User: question
Bot: Switches to question mode with instructions

User: How do I improve my coding skills?
Bot: Forwards to admin and confirms receipt

User: /cancel  
Bot: Returns to main menu

User: cv review
Bot: Shows CV review instructions

User: [shares Google Drive link]
Bot: Creates session and notifies admin
```

## 🚨 Error Handling

- **Wrong input:** Bot guides you back to correct options
- **Network issues:** Error messages help identify problems  
- **State confusion:** Use `/menu` or `/cancel` to reset
- **File problems:** Bot provides alternative upload methods

## 📊 Logging

The bot logs:
- **User entries:** Every user interaction is tracked
- **Errors:** All system errors for debugging
- **Commands used:** Which commands users prefer

Set `LOG_LEVEL=error` in `.env` for minimal logging (recommended).