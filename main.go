package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/NicoNex/echotron/v3"
	"github.com/jackc/pgx/v4/pgxpool"
)

type stateFn func(*echotron.Update) stateFn

type bot struct {
	chatID int64
	state  stateFn
	echotron.API
	db *pgxpool.Pool
}

var (
	telegramToken = os.Getenv("TELEGRAM_TOKEN")
	databaseUrl   = os.Getenv("DATABASE_URL")
)

func newBotWithDB(db *pgxpool.Pool, token string) echotron.NewBotFn {
	newBot := func(chatID int64) echotron.Bot {
		bot := &bot{
			chatID: chatID,
			API:    echotron.NewAPI(token),
			db:     db,
		}
		bot.state = bot.handleInitMessage
		return bot
	}

	return newBot
}

func (b *bot) Update(update *echotron.Update) {
	// Here we execute the current state and set the next one.
	b.state = b.state(update)
}

func (b *bot) handleInitMessage(update *echotron.Update) stateFn {

	b.API.SetMyCommands(
		&echotron.CommandOptions{LanguageCode: "en"},
		echotron.BotCommand{Command: "/start", Description: "Start the bot"},
		echotron.BotCommand{Command: "/babyinfo", Description: "Get your sugar baby's info"},
		echotron.BotCommand{Command: "/parent", Description: "Talk to your sugar parent"},
		echotron.BotCommand{Command: "/baby", Description: "Talk to your sugar baby"},
		echotron.BotCommand{Command: "/help", Description: "Get help"},
		echotron.BotCommand{Command: "/destroyplanet", Description: "Destroy the planet"})

	if update.Message == nil {
		return b.handleInitMessage
	}

	log.Println(update.Message.From.Username + " says: " + update.Message.Text)

	switch update.Message.Text {
	case "/start":
		return b.handleStart(update)
	default:
		b.SendMessage("Please /start the bot first.", b.chatID, nil)
		return b.handleInitMessage
	}
}

func (b *bot) handleMessage(update *echotron.Update) stateFn {

	if update.Message == nil {
		return b.handleMessage
	}

	log.Println(update.Message.From.Username + " says: " + update.Message.Text)

	switch update.Message.Text {
	case "/start":
		return b.handleStart(update)
	case "/parent":
		if !(isGameStarted(b.db)) {
			b.SendMessage("Sugar for Leo has not started. Please wait for update!!! 🥵", b.chatID, nil)
			return b.handleMessage
		}
		return b.handleMessageToParent(update)
	case "/baby":
		if !(isGameStarted(b.db)) {
			b.SendMessage("Sugar for Leo has not started. Please wait for update!!! 🥵", b.chatID, nil)
			return b.handleMessage
		}
		return b.handleMessageToBaby(update)
	case "/babyinfo":
		if !(isGameStarted(b.db)) {
			b.SendMessage("Sugar for Leo has not started. Please wait for update!!! 🥵", b.chatID, nil)
			return b.handleMessage
		}
		return b.handleBabyInfo(update)
	case "/admin":
		if update.Message.From.Username == "blur_sotong" || update.Message.From.Username == "edwinitis" || update.Message.From.Username == "itslengee" {
			return b.handleAdmin(update)
		} else {
			return b.handleMessage
		}
	case "/help":
		b.SendMessage("🆘For help with the bot, contact @blur_sotong. For questions on the Sugar for Leo event, contact Prog heads @edwinitis or @itslengee.", b.chatID, nil)
		return b.handleMessage
	case "/destroyplanet":
		b.SendMessage("🥵Feature coming soon... Stay tuned!", b.chatID, nil)
		return b.handleMessage
	case "penis":
		b.SendMessage("🤪The message is too long. Please try again.", b.chatID, nil)
		return b.handleMessage
	default:
		b.SendMessage("🤦You're not talking to anyone. Send /parent or /baby to start talking to your sugar parent or baby.", b.chatID, nil)
		return b.handleMessage
	}
}

func (b *bot) handleStart(update *echotron.Update) stateFn {

	if update.Message == nil {
		return b.handleMessage
	}

	username := update.Message.From.Username
	if !checkUser(b.db, username) {
		log.Println(update.Message.From.Username + " tried to start the bot.")
		b.SendMessage("You are not a registered participant. 😬😬😬 Please contact the house comm.", b.chatID, nil)
		return b.handleInitMessage
	}
	setChatID(b.db, username, b.chatID)
	log.Println(username + " started the bot, chatID: " + fmt.Sprint(b.chatID))
	if isGameStarted(b.db) {
		b.SendMessage("🦁🍬Welcome to Sugar for Leo bot! Send /babyinfo to get your sugar baby's info. Send /parent to talk to your sugar parent, /baby to talk to your sugar baby!", b.chatID, nil)
		return b.handleMessage
	} else {
		b.SendMessage("🦁🍬Welcome to Sugar for Leo bot! You have successfully started the bot. Please wait for Sugar for Leo to start!!!🤩", b.chatID, nil)
		return b.handleMessage
	}
}

func (b *bot) handleAdmin(update *echotron.Update) stateFn {

	if update.Message == nil {
		return b.handleAdmin
	}

	switch update.Message.Text {
	case "/admin":
		b.SendMessage("Entered admin mode. Send /players to get the list of current players. Send /unstartedplayers to check who haven't started the bot", b.chatID, nil)
		return b.handleAdmin
	case "/start":
		return b.handleStart(update)
	case "/done":
		b.SendMessage("Exited admin mode", b.chatID, nil)
		return b.handleMessage
	case "/resetandpopulatedata":
		devResetDB(b.db)
		initDB(b.db)
		populateDB(b.db)
		b.SendMessage("Data reset and populated", b.chatID, nil)
		return b.handleMessage
	case "/startsugarforleo":
		startGame(b.db)
		if isGameStarted(b.db) {
			b.SendMessage("Sugar for Leo started!", b.chatID, nil)
			chatids := getChatIDs(b.db)
			for _, chatid := range chatids {
				b.SendMessage("🎉🎉🎊🎊 <b>Sugar for Leo 22/23 has started!!!</b> 🎊🎊🎉🎉\n\nSend /babyinfo to get your sugar baby's info now! Send /parent or /baby to talk to your sugar parent or baby. Have fun!!! ( ͡° ͜ʖ ͡°)", chatid, &echotron.MessageOptions{ParseMode: "HTML"})
			}
		} else {
			b.SendMessage("OOPSIES Game could not be started!", b.chatID, nil)
		}
		b.SendMessage("Exited admin mode", b.chatID, nil)
		return b.handleMessage
	case "/stopsugarforleo":
		stopGame(b.db)
		if !isGameStarted(b.db) {
			b.SendMessage("Sugar for Leo stopped!", b.chatID, nil)
			chatids := getChatIDs(b.db)
			for _, chatid := range chatids {
				b.SendMessage("🎉🎉🎊🎊 <b>Sugar for Leo 22/23 has ended!!!</b> 🎊🎊🎉🎉\n\nThank you for your participation!!", chatid, &echotron.MessageOptions{ParseMode: "HTML"})
			}
		} else {
			b.SendMessage("OOPSIES Game could not be stopped!", b.chatID, nil)
		}
		b.SendMessage("Exited admin mode", b.chatID, nil)
		return b.handleMessage
	case "/devtogglestart":
		if isGameStarted(b.db) {
			stopGame(b.db)
			b.SendMessage("Sugar for Leo stopped!", b.chatID, nil)
		} else {
			startGame(b.db)
			b.SendMessage("Sugar for Leo started!", b.chatID, nil)
		}
		b.SendMessage("Exited admin mode", b.chatID, nil)
		return b.handleMessage
	case "/players":
		usernames := getAllUsernames(b.db)
		b.SendMessage("Current players:\n\n"+strings.Join(usernames, "\n"), b.chatID, nil)
		b.SendMessage("Exited admin mode", b.chatID, nil)
		return b.handleMessage
	case "/unstartedplayers":
		usernames := getUsersWithoutChatID(b.db)
		if len(usernames) == 0 {
			b.SendMessage("All players have started!", b.chatID, nil)
		} else {
			b.SendMessage("The following players have not started the bot:\n\n"+strings.Join(usernames, "\n"), b.chatID, nil)
		}
		b.SendMessage("Exited admin mode", b.chatID, nil)
		return b.handleMessage
	default:
		return b.handleAdmin
	}
}

func (b *bot) handleBabyInfo(update *echotron.Update) stateFn {

	if update.Message == nil {
		return b.handleMessage
	}

	username := update.Message.From.Username
	log.Println(username + " sent /babyinfo.")
	baby, err := getBaby(b.db, username)
	if err != nil {
		log.Println(err)
		return b.handleMessage
	}
	b.SendMessage("<b>Your sugar baby is</b> "+baby.name+", staying in "+baby.unit+"\n\n"+
		"<b>Tolerance level:</b> "+baby.level+"\n\n"+
		"❤️<b>Here are the likes of your sugar baby:</b>❤️"+"\n"+baby.likes+"\n\n"+
		"👎<b>Here are the dislikes of your sugar baby:</b>👎"+"\n"+baby.dislikes+"\n\n"+
		"❌<b>These are your sugar baby's no-gos:</b>❌"+"\n"+baby.nogos+"\n\n"+
		"<b>Please take these remarks seriously:</b> \n"+baby.remarks+"\n\n"+
		"Send /baby to start talking to your sugar baby!!", b.chatID, &echotron.MessageOptions{ParseMode: "HTML"})
	return b.handleMessage
}

func (b *bot) handleMessageToParent(update *echotron.Update) stateFn {

	if update.Message == nil {
		return b.handleMessage
	}

	username := update.Message.From.Username
	log.Println(username + " says to parent: " + update.Message.Text)
	parent, err := getParent(b.db, username)
	if err != nil {
		log.Println(err)
		return b.handleMessage
	}

	if !parent.chatID.Valid {
		b.SendMessage("😔Your sugar parent has not started the bot. Please contact the house comm", b.chatID, nil)
		return b.handleMessage
	}

	switch update.Message.Text {
	case "/start":
		return b.handleStart(update)
	case "/done":
		b.SendMessage("You are no longer talking to your sugar parent.", b.chatID, nil)
		return b.handleMessage
	case "/parent":
		b.SendMessage("🎅You are now talking to your sugar parent. Send /done to finish. You can send multiple messages at a time, including photos, videos and stickers. (Try not to spam🥵)", b.chatID, nil)
		return b.handleMessageToParent
	case "/baby":
		return b.handleMessageToBaby(update)
	case "/babyinfo":
		return b.handleMessage(update)
	case "/help":
		return b.handleMessage(update)
	case "/destroyplanet":
		return b.handleMessage(update)
	case "/admin":
		return b.handleMessage(update)
	case "penis":
		b.SendMessage("🤪The message is too long. Please try again.", b.chatID, nil)
		return b.handleMessageToParent
	}

	if update.Message.Sticker != nil {
		b.SendMessage("👶Sticker from sugar baby:", parent.chatID.Int64, nil)
		b.SendSticker(update.Message.Sticker.FileID, parent.chatID.Int64, nil)
	}
	if update.Message.Photo != nil {
		b.SendPhoto(echotron.NewInputFileID(update.Message.Photo[len(update.Message.Photo)-1].FileID), parent.chatID.Int64, &echotron.PhotoOptions{Caption: "👶Photo from sugar baby"})
	}
	if update.Message.Video != nil {
		b.SendVideo(echotron.NewInputFileID(update.Message.Video.FileID), parent.chatID.Int64, &echotron.VideoOptions{Caption: "👶Video from sugar baby"})
	}
	if update.Message.Text != "" {
		b.SendMessage("👶Sugar baby:\n"+update.Message.Text, parent.chatID.Int64, nil)
	}
	return b.handleMessageToParent
}

func (b *bot) handleMessageToBaby(update *echotron.Update) stateFn {

	if update.Message == nil {
		return b.handleMessage
	}

	username := update.Message.From.Username
	log.Println(username + " says to baby: " + update.Message.Text)
	baby, err := getBaby(b.db, username)
	if err != nil {
		log.Println(err)
		return b.handleMessage
	}

	if !baby.chatID.Valid {
		b.SendMessage("😔Your sugar baby has not started the bot. Please contact the house comm", b.chatID, nil)
		return b.handleMessage
	}

	switch update.Message.Text {
	case "/start":
		return b.handleStart(update)
	case "/done":
		b.SendMessage("You are no longer talking to your sugar baby.", b.chatID, nil)
		return b.handleMessage
	case "/baby":
		b.SendMessage("👶You are now talking to your sugar baby. Send /done to finish. You can send multiple messages at a time, including photos, videos and stickers. (Try not to spam🥵)", b.chatID, nil)
		return b.handleMessageToBaby
	case "/parent":
		return b.handleMessageToParent(update)
	case "/babyinfo":
		return b.handleMessage(update)
	case "/help":
		return b.handleMessage(update)
	case "/destroyplanet":
		return b.handleMessage(update)
	case "/admin":
		return b.handleMessage(update)
	case "penis":
		b.SendMessage("🤪The message is too long. Please try again.", b.chatID, nil)
		return b.handleMessageToBaby
	}

	if update.Message.Sticker != nil {
		b.SendMessage("🎅Sticker from sugar parent:", baby.chatID.Int64, nil)
		b.SendSticker(update.Message.Sticker.FileID, baby.chatID.Int64, nil)
	}
	if update.Message.Photo != nil {
		b.SendPhoto(echotron.NewInputFileID(update.Message.Photo[len(update.Message.Photo)-1].FileID), baby.chatID.Int64, &echotron.PhotoOptions{Caption: "🎅Photo from sugar parent"})
	}
	if update.Message.Video != nil {
		b.SendVideo(echotron.NewInputFileID(update.Message.Video.FileID), baby.chatID.Int64, &echotron.VideoOptions{Caption: "🎅Video from sugar parent"})
	}
	if update.Message.Text != "" {
		b.SendMessage("🎅Sugar parent:\n"+update.Message.Text, baby.chatID.Int64, nil)
	}
	return b.handleMessageToBaby
}

// env GOOS=linux GOARCH=amd64 go build -o ./bin/sugarforleo
func main() {
	db := connectDB()
	defer db.Close()

	initDB(db)
	populateDB(db)

	dsp := echotron.NewDispatcher(telegramToken, newBotWithDB(db, telegramToken))
	log.Println("Bot running...")
	log.Println(dsp.Poll())
}
