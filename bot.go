package main

import (
	"fmt"
	"log"
	"time"
	"math"
	"errors"
	"strconv"

	"github.com/line/line-bot-sdk-go/linebot"
)

var failedMsg = "Failed to process command from user with ID %s"

func connectChannel(bot *linebot.Client, token, data, userID string) (err error) {
	var args []string

	args, err = parse(data)
	if len(args) != 2 || err != nil {
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "connect"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}
	
	var channel Channel
	db.Find(&channel, "name = ? AND key = ?", args[0], args[1])

	if channel.Name == "" && channel.Key == "" {
		sendMessage(bot, token, "There is no channel with that name & key.")
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	var user User
	db.Find(&user, "id = ?", userID)
	if user.ChannelUser == "" {
		db.Create(&User{ID: userID, ChannelUser: channel.Name})
	} else {
		db.Model(&user).Update("channel_user", channel.Name)
	}

	sendMessage(bot, token, fmt.Sprintf(connectMsg, args[0]))
	return
}

func createChannel(bot *linebot.Client, token, data, userID string) (err error) {
	var args []string

	args, err = parse(data)
	if err != nil || len(args) != 2 || len(args[0]) == 0 || len(args[1]) == 0 {
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "create channel"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	res := db.Create(&Channel{Name: args[0], Key: args[1]})
	if res.Error != nil {
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "create channel"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	sendMessage(bot, token, fmt.Sprintf(createChannelMsg, args[0]))
	return
}

func listTasks(bot *linebot.Client, token, data, userID string) (err error) {
	var args []string

	args, err = parse(data)
	if err != nil || (len(args) != 1 && len(args) != 2) {
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "tasks"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	var offset int
	if len(args) == 2 {
		offset, err = strconv.Atoi(args[1])
		if err != nil {
			sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "tasks"))
			err = errors.New(fmt.Sprintf(failedMsg, userID))
			return
		}
	}

	var user User
	db.Find(&user, "id = ?", userID)
	if user.ID == "" || user.ChannelUser == "" {
		sendMessage(bot, token, connectFirstMsg)
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	location, _ := time.LoadLocation("Asia/Jakarta")
	var tasks []Task
	var query string
	switch args[0] {
		case "now":
			query = "channel_task = ? AND date >= ?"
		case "past":
			query = "channel_task = ? AND date < ?"
		default:
			sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "tasks"))
			err = errors.New(fmt.Sprintf(failedMsg, userID))
			return
	}

	var counts int64
	now := time.Now().In(location)
	db.Model(&Task{}).Where(query, user.ChannelUser, now).Count(&counts)
	if counts <= int64(offset*9) {
		sendMessage(bot, token, fmt.Sprintf("Page %d is not available", offset))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}
	db.Order("date").Limit(9).Offset(offset*9).Find(&tasks, query, user.ChannelUser, now)

	msg := "Swipe to the left >>"
	if len(tasks) == 0 {
		msg = "There is no task yet"
	}

	contents := []*linebot.BubbleContainer{createBubbleTaskTitle(uint(offset+1), uint(math.Ceil(float64(counts)/9.0)), msg)}
	for _, task := range tasks {
		contents = append(contents, createBubbleTask(task))
	}

	container := &linebot.CarouselContainer{
		Type: linebot.FlexContainerTypeCarousel,
		Contents: contents,
	}

	_, err = bot.ReplyMessage(token, linebot.NewFlexMessage("Channel's Tasks List", container)).Do();
	if err != nil {
		log.Print(err)
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}
	return
}

func addTask(bot *linebot.Client, token, data, userID string) (err error) {
	var args []string

	args, err = parse(data)
	if err != nil || len(args) != 3 || len(args[0]) == 0 || len(args[1]) == 0 {
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "add task"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	var user User
	res := db.Find(&user, "id = ?", userID)
	if user.ID == "" || user.ChannelUser == "" {
		sendMessage(bot, token, connectFirstMsg)
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	var t time.Time
	t, err = time.Parse("2006-01-02 15:04:05 MST -07:00", args[1]+" GMT +07:00")
	if err != nil {
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "add task"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
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
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "add task"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	sendMessage(bot, token, fmt.Sprintf(addTaskMsg, args[0]))
	return
}

func deleteTask(bot *linebot.Client, token, data, userID string) (err error) {
	length := len(data)
	if length < 15 {
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "delete task"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	var id int
	id, err = strconv.Atoi(data[14:length-1])
	if err != nil {
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "delete task"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	var user User
	res := db.Find(&user, "id = ?", userID)
	if user.ID == "" || user.ChannelUser == "" {
		sendMessage(bot, token, connectFirstMsg)
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	var task Task
	res = db.Where("channel_task = ? AND task_id = ?", user.ChannelUser, id).Delete(&task)
	if res.Error != nil {
		sendMessage(bot, token, fmt.Sprintf(commandFailedMsg, "delete task"))
		err = errors.New(fmt.Sprintf(failedMsg, userID))
		return
	}

	sendMessage(bot, token, fmt.Sprintf(deleteTaskMsg, task.Name))
	return
}