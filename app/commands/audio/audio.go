package audio

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/jonas747/dca"
	handler "quozlet.net/birbbot/util"
)

// Data contains encoded audio session and the author that initiated its creation
type Data struct {
	audio          *audioSource
	mutex          *sync.Mutex
	VoiceChannelID string
	TextChannelID  string
	GuildID        string
	Title          string
}

type audioSource struct {
	filename string
	session  *dca.EncodeSession
}

// CacheAsFile takes the current encoding session and copies it to a file
// This is to reduce "unnecessary" memory consumption, and should be done eagerly
func (d Data) CacheAsFile() {
	tmpFile, err := ioutil.TempFile("", "*.dca")
	if err != nil {
		log.Printf("Failed to cache stream as file: %s", err)
		return
	}
	defer handler.LogErrorMsg("Failed to close temp file: %s", tmpFile.Close())
	d.mutex.Lock()
	_, err = io.Copy(tmpFile, d.audio.session)
	if err != nil {
		handler.LogErrorMsg("Failed to copy, aborting cache: %s", err)
		handler.LogErrorMsg("Failed to remove temp file: %s", os.Remove(d.audio.filename))
		d.mutex.Unlock()
		return
	}
	d.audio.session = nil
	d.audio.filename = tmpFile.Name()
	d.mutex.Unlock()
	log.Printf("Cached %s as a file", d.Title)
}

// AudioSource fetches the audio source
// If it has been cached as a file, it fetches from the file
// The data will be unable to be modified until cleanup
func (d Data) AudioSource() (*dca.EncodeSession, *dca.Decoder, error) {
	d.mutex.Lock()
	if d.audio.session == nil {
		file, err := os.Open(d.audio.filename)
		if err != nil {
			return nil, nil, err
		}
		return nil, dca.NewDecoder(file), nil
	}
	return d.audio.session, nil, nil
}

// Cleanup function that removes the temporary file, if it exists
func (d Data) Cleanup() error {
	d.mutex.Unlock()
	d.mutex.Lock()
	if d.audio.session == nil {
		return os.Remove(d.audio.filename)
	}
	d.audio.session = nil
	d.mutex.Unlock()
	return nil
}

// VoiceCommand indicates a special handling command besides playing audio
type VoiceCommand int

const (
	// Leave will remove voice and clean the queue
	Leave VoiceCommand = iota
	// Start will play if the stream was paused
	Start
	// Stop will stop if the stream is playing
	Stop
)

var inVoice = false
var mutex = &sync.Mutex{}

// SetInVoice sets the global state of whether audio is playing or not
func SetInVoice(state bool) {
	mutex.Lock()
	inVoice = state
	mutex.Unlock()
}

// IsInVoiceChannel returns the global state of whether audio is playing or not
func IsInVoiceChannel() bool {
	var currentState bool
	mutex.Lock()
	currentState = inVoice
	mutex.Unlock()
	return currentState
}
