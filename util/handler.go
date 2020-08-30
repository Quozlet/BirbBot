package handler

import (
	"log"
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
