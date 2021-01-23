package main

import (
	"log"
	"os"
	"io/ioutil"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	welcomeMsg = "Dizappl - Commands List."
	commandFailedMsg = "Command %s has failed."
	commandNotFoundMsg = "Command Not Found. Please use `!help` to retrieve commands list!"
	connectFirstMsg = "Please connect/create channel first before using other commands."
	connectMsg = "You have successfully connected to channel %s."
	createChannelMsg = "Channel %s has been created."
	addTaskMsg = "Task %s has been added."
	deleteTaskMsg = "Task %s has been deleted."

	db = connectToDB("/dizappl.db")
)

func main() {
	f, _ := os.Open("key.json")
	data, _ := ioutil.ReadAll(f)
	var creds map[string]string
	json.Unmarshal(data, &creds)
	bot, err := linebot.New(creds["channel_secret"], creds["channel_token"])
	check(err)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		for _, event := range events {
			var err error
			if event.Type == linebot.EventTypeMessage {
				userID := event.Source.UserID
				switch message := event.Message.(type) {
					case *linebot.TextMessage:
						msgLength := len(message.Text)
						if msgLength > 1 && message.Text[0] == '!' {
							if strings.EqualFold(message.Text, "!help") {
								container, _ := linebot.UnmarshalFlexMessageJSON(getMessage("welcome"))
								_, err = bot.ReplyMessage(event.ReplyToken, linebot.NewFlexMessage(welcomeMsg, container)).Do();
								if err != nil {
									log.Print(err)
								}
							} else if msgLength > 9 && strings.EqualFold(message.Text[:9], "!connect ") {
								err = connectChannel(bot, event.ReplyToken, message.Text[9:], userID)
								if err != nil {
									log.Print(err)
								}
							} else if msgLength > 16 && strings.EqualFold(message.Text[:16], "!create channel ") {
								err = createChannel(bot, event.ReplyToken, message.Text[16:], userID)
								if err != nil {
									log.Print(err)
								}
							} else if msgLength > 7 && strings.EqualFold(message.Text[:7], "!tasks ") {
								err = listTasks(bot, event.ReplyToken, message.Text[7:], userID)
								if err != nil {
									log.Print(err)
								}
							} else if msgLength > 10 && strings.EqualFold(message.Text[:10], "!add task ") {
								err = addTask(bot, event.ReplyToken, message.Text[10:], userID)
								if err != nil {
									log.Print(err)
								}
							} else if msgLength > 13 && strings.EqualFold(message.Text[:13], "!delete task ") {
								err = deleteTask(bot, event.ReplyToken, message.Text, userID)
								if err != nil {
									log.Print(err)
								}
							} else {
								sendMessage(bot, event.ReplyToken, commandNotFoundMsg)
							}
						}
				}
			} else if (event.Type == linebot.EventTypeJoin || event.Type == linebot.EventTypeFollow) {
				container, _ := linebot.UnmarshalFlexMessageJSON(getMessage("welcome"))
				_, err = bot.ReplyMessage(event.ReplyToken, linebot.NewFlexMessage(welcomeMsg, container)).Do();
				if err != nil {
					log.Print(err)
				}
			}
		}
	})

	if err := http.ListenAndServe(":5000", nil); err != nil {
		panic(err)
	}
}