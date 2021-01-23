package main

import (
	"fmt"
	"os"
	"io"
	"log"
	"io/ioutil"
	"errors"

	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
	"github.com/line/line-bot-sdk-go/linebot"
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

func createBubbleTaskTitle(page, totalPages uint, msg string) *linebot.BubbleContainer {
	return &linebot.BubbleContainer{
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
					Text: fmt.Sprintf("Page %d of %d", page, totalPages),
					Wrap: true,
					Size: "xs",
				},
				&linebot.TextComponent{
					Type: linebot.FlexComponentTypeText,
					Wrap: true,
					Text: msg,
				},
			},
		},
	}
}

func createBubbleTask(task Task) *linebot.BubbleContainer {
	return &linebot.BubbleContainer{
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
	}
}