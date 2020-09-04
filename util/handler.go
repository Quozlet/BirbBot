package handler

import (
	"fmt"
	"log"
	"strings"

	"quozlet.net/birbbot/app/commands"
)

// LogError takes in an error and logs if not nil
func LogError(err error) {
	if err != nil {
		log.Println(err)
	}
}

// LogErrorMsg takes in a message and error
// The custom message is logged if the error message is not nil
func LogErrorMsg(msg string, err error) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

// SendErrorMsg sends a message if the error is not nil
func SendErrorMsg(msg commands.MessageResponse, messageChannel chan<- commands.MessageResponse, err error) bool {
	if err != nil {
		log.Println(err)
		if strings.Contains(msg.Message, "%s") {
			msg.Message = fmt.Sprintf(msg.Message, err)
		}
		messageChannel <- msg
		return true
	}
	return false
}
