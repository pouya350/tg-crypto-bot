# Telegram Crypto & Risk Management Bot

A robust Telegram bot built with Go, implementing a Finite State Machine (FSM) for seamless multi-step user interactions. The bot provides real-time cryptocurrency price tracking via the Binance API and features a built-in risk management calculator for trading.

## 🚀 Features
* **Finite State Machine (FSM)**: Strict state management for complex dialogs and command routing.
* **Binance API Integration**: Real-time cryptocurrency price parsing using `net/http` and `encoding/json`.
* **Risk Management Calculator**: Step-by-step mathematical calculation of trade risks, deposits, and stop-losses.
* **Secure Configuration**: Environment variables management using `godotenv` to protect API keys and database credentials.
* **Clean Architecture**: Codebase structured with separation of concerns (`cmd/` for entrypoints, `internal/` for logic and storage).
* **Dynamic Routing**: Scalable command and status map handlers for easy navigation.

## 🛠 Tech Stack
* **Language**: Go (Golang)
* **Telegram API**: `github.com/go-telegram-bot-api/telegram-bot-api/v5`
* **Database**: PostgreSQL (with custom storage wrapper)
* **External API**: Binance Public API

## ⚙️ Setup and Installation

