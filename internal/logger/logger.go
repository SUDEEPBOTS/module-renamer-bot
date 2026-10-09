package logger

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type EventLogger struct {
	enabled    int32
	botAPI     *tgbotapi.BotAPI
	logChannel int64
}

var GlobalLogger = &EventLogger{
	enabled: 1,
}

func InitLogger(bot *tgbotapi.BotAPI, channelID int64) {
	GlobalLogger.botAPI = bot
	GlobalLogger.logChannel = channelID
	atomic.StoreInt32(&GlobalLogger.enabled, 1)
}

func SetEnabled(enable bool) {
	if enable {
		atomic.StoreInt32(&GlobalLogger.enabled, 1)
		LogInfo("SYSTEM", "Event logging enabled by administrator")
	} else {
		LogInfo("SYSTEM", "Event logging disabled by administrator")
		atomic.StoreInt32(&GlobalLogger.enabled, 0)
	}
}

func IsEnabled() bool {
	return atomic.LoadInt32(&GlobalLogger.enabled) == 1
}

func formatTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func LogInfo(tag, message string) {
	if !IsEnabled() {
		return
	}
	line := fmt.Sprintf("[%s] [INFO] [%s] %s", formatTimestamp(), tag, message)
	log.Println(line)
}

func LogError(tag string, err error) {
	line := fmt.Sprintf("[%s] [ERROR] [%s] %v", formatTimestamp(), tag, err)
	log.Println(line)
}

func LogUserStart(userID int64, username, firstName string) {
	if !IsEnabled() {
		return
	}
	msg := fmt.Sprintf("USER_START: UserID=%d Username=%s Name=%s", userID, username, firstName)
	line := fmt.Sprintf("[%s] [INFO] %s", formatTimestamp(), msg)
	log.Println(line)

	dispatchToChannel(fmt.Sprintf("[AUDIT] USER_START\nUserID: %d\nUsername: @%s\nName: %s\nTimestamp: %s", userID, username, firstName, formatTimestamp()))
}

func LogRenameInitiated(userID int64, username, source, oldName, newName string) {
	if !IsEnabled() {
		return
	}
	msg := fmt.Sprintf("RENAME_INITIATED: UserID=%d Username=%s Source=%s Old=%s New=%s", userID, username, source, oldName, newName)
	line := fmt.Sprintf("[%s] [INFO] %s", formatTimestamp(), msg)
	log.Println(line)

	dispatchToChannel(fmt.Sprintf("[AUDIT] RENAME_INITIATED\nUserID: %d\nUsername: @%s\nSource: %s\nRebranding: %s -> %s\nTimestamp: %s", userID, username, source, oldName, newName, formatTimestamp()))
}

func LogRenameSuccess(userID int64, username, oldName, newName string, scanned, modified, replacements, paths int) {
	if !IsEnabled() {
		return
	}
	msg := fmt.Sprintf("RENAME_SUCCESS: UserID=%d Username=%s Rebranded: %s -> %s | Scanned=%d Modified=%d Replacements=%d Paths=%d", userID, username, oldName, newName, scanned, modified, replacements, paths)
	line := fmt.Sprintf("[%s] [INFO] %s", formatTimestamp(), msg)
	log.Println(line)

	dispatchToChannel(fmt.Sprintf("[AUDIT] RENAME_SUCCESS\nUserID: %d\nUsername: @%s\nRebranding: %s -> %s\nScanned Files: %d\nModified Files: %d\nReplacements: %d\nRenamed Paths: %d\nTimestamp: %s", userID, username, oldName, newName, scanned, modified, replacements, paths, formatTimestamp()))
}

func LogRenameFailure(userID int64, username string, err error) {
	msg := fmt.Sprintf("RENAME_FAILED: UserID=%d Username=%s Error=%v", userID, username, err)
	line := fmt.Sprintf("[%s] [ERROR] %s", formatTimestamp(), msg)
	log.Println(line)

	dispatchToChannel(fmt.Sprintf("[AUDIT] RENAME_FAILED\nUserID: %d\nUsername: @%s\nError: %v\nTimestamp: %s", userID, username, err, formatTimestamp()))
}

func dispatchToChannel(text string) {
	if GlobalLogger.botAPI != nil && GlobalLogger.logChannel != 0 && IsEnabled() {
		go func() {
			msg := tgbotapi.NewMessage(GlobalLogger.logChannel, "<pre>"+text+"</pre>")
			msg.ParseMode = "HTML"
			_, _ = GlobalLogger.botAPI.Send(msg)
		}()
	}
}

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
}
