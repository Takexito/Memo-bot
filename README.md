# Memo Bot

A Telegram bot that helps you organize and classify your notes, images, and files automatically. The bot analyzes the content of your messages and assigns relevant tags for easy retrieval later.

## Features

- Automatically classifies and tags your content using ChatGPT
- Supports multiple content types:
  - Text messages
  - Images with captions
  - Documents
  - Videos
- Intelligent tag generation using OpenAI's GPT model
- Easy note retrieval by tags
- PostgreSQL storage for persistence
- Fallback to simple classification if GPT is unavailable
- Easy deployment to Vercel

## Prerequisites

- Go 1.21 or later
- PostgreSQL database (or use a managed service like Supabase)
- Telegram Bot Token (from [@BotFather](https://t.me/botfather))
- OpenAI API Key (for ChatGPT integration)
- Vercel account (for deployment)

## Setup

1. Clone the repository:
```bash
git clone https://github.com/xaenox/memo-bot.git
cd memo-bot
```

2. Install dependencies:
```bash
go mod download
```

3. Create the PostgreSQL database and tables:
```sql
CREATE DATABASE memo_bot;

\c memo_bot

CREATE TABLE notes (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content TEXT,
    type VARCHAR(20) NOT NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    file_id TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_notes_user_id ON notes(user_id);
CREATE INDEX idx_notes_tags ON notes USING GIN(tags);
```

4. Configure the bot:
   - Copy `config.yaml` to your working directory
   - Update the configuration with your:
     - Telegram Bot Token
     - PostgreSQL credentials
     - OpenAI API Key and settings
     - Classification settings

## Configuration

Edit `config.yaml` with your settings:

```yaml
telegram:
  token: "YOUR_BOT_TOKEN"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "memo_bot"
  sslmode: "disable"

classifier:
  min_confidence: 0.7
  max_tags: 5

openai:
  api_key: "YOUR_OPENAI_API_KEY"  # Get this from https://platform.openai.com/api-keys
  model: "gpt-3.5-turbo"         # GPT model to use
  max_tokens: 150                # Maximum tokens for response
  temperature: 0.3               # Lower values for more focused/deterministic responses
```

### Setting up OpenAI API

1. Create an OpenAI account at https://platform.openai.com/signup
2. Go to https://platform.openai.com/api-keys
3. Create a new API key
4. Copy the API key and paste it in your `config.yaml` file
5. (Optional) Adjust the model and parameters:
   - `model`: Choose between available models (e.g., "gpt-3.5-turbo", "gpt-4")
   - `max_tokens`: Adjust based on your needs (higher values = longer responses)
   - `temperature`: Adjust between 0-1 (lower = more focused, higher = more creative)

## Deployment to Vercel

### Prerequisites for Vercel Deployment
1. Install Vercel CLI:
```bash
npm install -g vercel
```

2. Login to Vercel:
```bash
vercel login
```

### Environment Variables
Set up the following environment variables in your Vercel project settings:
- `TELEGRAM_TOKEN`: Your Telegram Bot Token
- `OPENAI_API_KEY`: Your OpenAI API Key
- `DATABASE_URL`: Your PostgreSQL connection string
- `MAX_TAGS`: Maximum number of tags (e.g., "5")
- `MIN_CONFIDENCE`: Minimum confidence score (e.g., "0.7")

### Deployment Steps
1. Push your code to GitHub
2. Link your GitHub repository to Vercel:
```bash
vercel link
```

3. Deploy to Vercel:
```bash
vercel deploy
```

4. (Optional) Set up automatic deployments:
   - Go to your Vercel dashboard
   - Select your project
   - Enable "Git Integration"
   - Choose your repository
   - Configure build settings

### Database Setup for Production
For production, we recommend using a managed PostgreSQL service:
1. Create a database on a service like Supabase, Railway, or AWS RDS
2. Update the `DATABASE_URL` environment variable in Vercel
3. Run the database migration script (provided in the setup section)

## Running Locally

```bash
go run cmd/bot/main.go
```

## Usage

1. Start a chat with your bot on Telegram
2. Send `/start` to get started
3. Send any message, photo, or document
4. The bot will use ChatGPT to analyze your content and generate relevant tags
5. Use `/list` to see your recent notes
6. Use `/list #tag` to filter notes by tag

## Commands

- `/start` - Start the bot
- `/help` - Show help message
- `/list` - List your recent notes
- `/list #tag` - List notes with specific tag

## How Tag Generation Works

The bot uses ChatGPT to analyze your content and generate relevant tags by:
1. Understanding the semantic meaning of your messages
2. Identifying key themes and topics
3. Extracting relevant keywords and concepts
4. Maintaining consistency in tag naming

If the ChatGPT API is unavailable, the bot automatically falls back to a simple classification system based on keywords and hashtags.

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details. 