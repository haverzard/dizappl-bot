package main

import (
	"fmt"
	"log"
	"os"
	"io"
	"io/ioutil"
	"encoding/json"
	"net/http"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
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

	db = connectToDB("dizappl.db")
)

func connectToDB(name string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(name), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	doMigration(db)
	return db
}


func sendMessage(bot *linebot.Client, replyToken string, msg string) {
	_, err := bot.ReplyMessage(replyToken, linebot.NewTextMessage(msg)).Do();
	if err != nil {
		log.Print(err)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func getMessage(name string) []byte {
	var (
		err error
		f io.Reader
		data []byte
	)
	f, err = os.Open(fmt.Sprintf("messages/%s.json", name))
	check(err)
	data, err = ioutil.ReadAll(f)
	check(err)
	return data
}

func parse(data string) (args []string, err error) {
	var (
		length = 0
		parser = 0
		buf = ""
	)
	for _, c := range data {
		if parser == 1 {
			if c == '\'' {
				if length != 0 && buf[length-1] == '\\' {
					buf = buf[:length-1]
					length--
				} else {
					parser = 2
					args = append(args, buf)
					buf = ""
					length = 0
				}
			}
			if parser == 1 {
				buf += string(c)
				length++
			}
		} else {
			if parser == 2 {
				if c != ' ' {
					err = errors.New("Data is invalid")
					return
				}
				parser = 0
			} else if c == '\'' {
				parser = 1
			} else {
				err = errors.New("Data is invalid")
				return
			}
		}
	}
	if buf != "" {
		err = errors.New("Data is invalid")
	}
	return
}

func main() {
	f, _ := os.Open("key.json")
	data, _ := ioutil.ReadAll(f)
	var creds map[string]interface{}
	creds := json.Unmarshal(data, &creds)
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
								var args []string

								args, err = parse(message.Text[9:])
								if len(args) != 2 || err != nil {
									sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "connect"))
									break
								}
								
								var channel Channel
								db.Find(&channel, "name = ? AND key = ?", args[0], args[1])

								if channel.Name == "" && channel.Key == "" {
									sendMessage(bot, event.ReplyToken, "There is no channel with that name & key.")
									break
								}

								var user User
								db.Find(&user, "id = ?", userID)
								if user.ChannelUser == "" {
									db.Create(&User{ID: userID, ChannelUser: channel.Name})
								} else {
									db.Model(&user).Update("channel_user", channel.Name)
								}

								sendMessage(bot, event.ReplyToken, fmt.Sprintf(connectMsg, args[0]))
							} else if msgLength > 16 && strings.EqualFold(message.Text[:16], "!create channel ") {
								var args []string

								args, err = parse(message.Text[16:])
								if err != nil || len(args) != 2 || len(args[0]) == 0 || len(args[1]) == 0 {
									sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "create channel"))
									break
								}

								res := db.Create(&Channel{Name: args[0], Key: args[1]})
								if res.Error != nil {
									sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "create channel"))
									break
								}

								sendMessage(bot, event.ReplyToken, fmt.Sprintf(createChannelMsg, args[0]))
							} else if msgLength > 7 && strings.EqualFold(message.Text[:7], "!tasks ") {
								var user User
								db.Find(&user, "id = ?", userID)
								if user.ID == "" || user.ChannelUser == "" {
									sendMessage(bot, event.ReplyToken, connectFirstMsg)
									break
								}

								location, _ := time.LoadLocation("Asia/Jakarta")
								var tasks []Task
								var query string
								switch message.Text[7:] {
									case "now":
										query = "channel_task = ? AND date >= ?"
									case "past":
										query = "channel_task = ? AND date < ?"
									default:
										sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "tasks"))
										break
								}
								// var counts int64
								db.Order("date").Limit(9).Find(&tasks, query, user.ChannelUser, now)

								msg := "Swipe to the left >>"
								if len(tasks) == 0 {
									msg = "There is no task yet"
								}

								contents := []*linebot.BubbleContainer{
									&linebot.BubbleContainer{
										Type: linebot.FlexContainerTypeBubble,
										Body: &linebot.BoxComponent{
											Type:   linebot.FlexComponentTypeBox,
											Layout: linebot.FlexBoxLayoutTypeVertical,
											Contents: []linebot.FlexComponent{
												&linebot.TextComponent{
													Type: linebot.FlexComponentTypeText,
													Text: "Tasks",
													Weight: "bold",
													Size: "sm",
													Wrap: true,
												},
												&linebot.TextComponent{
													Type: linebot.FlexComponentTypeText,
													Wrap: true,
													Text: msg,
												},
											},
										},
									},
								}
								for _, task := range tasks {
									contents = append(contents, &linebot.BubbleContainer{
										Type: linebot.FlexContainerTypeBubble,
										Body: &linebot.BoxComponent{
											Type:   linebot.FlexComponentTypeBox,
											Layout: linebot.FlexBoxLayoutTypeVertical,
											Contents: []linebot.FlexComponent{
												&linebot.TextComponent{
													Type: linebot.FlexComponentTypeText,
													Text: task.Name,
													Wrap: true,
													Weight: "bold",
													Size: "sm",
												},
												&linebot.TextComponent{
													Type: linebot.FlexComponentTypeText,
													Text: fmt.Sprintf("ID: %d", task.TaskID),
													Wrap: true,
													Size: "xs",
												},
												&linebot.TextComponent{
													Type: linebot.FlexComponentTypeText,
													Text: task.Date.Format("Mon Jan 2 2006, 15:04:05"),
													Color: "#8c8c8c",
													Wrap: true,
													Size: "xs",
												},
												&linebot.TextComponent{
													Type: linebot.FlexComponentTypeText,
													Text: task.Description,
													Wrap: true,
													Size: "xs",
												},
											},
										},
									})
								}

								container := &linebot.CarouselContainer{
									Type: linebot.FlexContainerTypeCarousel,
									Contents: contents,
								}

								_, err = bot.ReplyMessage(event.ReplyToken, linebot.NewFlexMessage("Channel's Tasks List", container)).Do();
								if err != nil {
									log.Print(err)
								}
							} else if msgLength > 10 && strings.EqualFold(message.Text[:10], "!add task ") {
								var args []string

								args, err = parse(message.Text[10:])
								if err != nil || len(args) != 3 || len(args[0]) == 0 || len(args[1]) == 0 {
									sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "add task"))
									break
								}

								var user User
								res := db.Find(&user, "id = ?", userID)
								if user.ID == "" || user.ChannelUser == "" {
									sendMessage(bot, event.ReplyToken, connectFirstMsg)
									break
								}

								var t time.Time
								t, err = time.Parse("2006-01-02 15:04:05 MST -07:00", args[1]+" GMT +07:00")
								if err != nil {
									sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "add task"))
									break
								}

								var task Task
								var count uint
								res = db.Where("channel_task = ?", user.ChannelUser).Last(&task)
								count = task.TaskID
								if res.Error != nil {
									count = 0
								}

								res = db.Create(&Task{TaskID: count+1, Name: args[0], Date: t, Description: args[2], ChannelTask: user.ChannelUser})
								if res.Error != nil {
									sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "add task"))
									break
								}

								sendMessage(bot, event.ReplyToken, fmt.Sprintf(addTaskMsg, args[0]))
							} else if msgLength > 13 && strings.EqualFold(message.Text[:13], "!delete task ") {
								length := len(message.Text)
								if length < 15 {
									sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "delete task"))
									break
								}

								var id int
								id, err = strconv.Atoi(message.Text[14:length-1])
								if err != nil {
									sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "delete task"))
									break
								}

								var user User
								res := db.Find(&user, "id = ?", userID)
								if user.ID == "" || user.ChannelUser == "" {
									sendMessage(bot, event.ReplyToken, connectFirstMsg)
									break
								}

								var task Task
								res = db.Where("channel_task = ? AND task_id = ?", user.ChannelUser, id).Delete(&task)
								if res.Error != nil {
									sendMessage(bot, event.ReplyToken, fmt.Sprintf(commandFailedMsg, "delete task"))
									break
								}

								sendMessage(bot, event.ReplyToken, fmt.Sprintf(deleteTaskMsg, task.Name))
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