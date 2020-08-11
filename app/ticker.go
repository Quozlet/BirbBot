package app

import (
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"quozlet.net/birbbot/app/commands/recurring"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Timers at which RecurringCommand intervals are supported
type Timers struct {
	Daily           *time.Ticker
	Hourly          *time.Ticker
	Minutely        *time.Ticker
	FiveMinutely    *time.Ticker
	TenMinutely     *time.Ticker
	QuarterHourly   *time.Ticker
	HalfHourly      *time.Ticker
	QuarterToHourly *time.Ticker
}

// Start looking for new messages to post at all the supported intervals
func (t Timers) Start(recurringCommandMap map[recurring.Frequency][]*RecurringCommand, dbPool *pgxpool.Pool, session *discordgo.Session) {
	go func() {
		for {
			select {
			case <-t.Daily.C:
				if len(recurringCommandMap[recurring.Daily]) != 0 {
					log.Println("Daily check ran")
					processRecurringMsg(recurringCommandMap[recurring.Daily], dbPool, session)
				}
			case <-t.Hourly.C:
				if len(recurringCommandMap[recurring.Hourly]) != 0 {
					log.Println("Hourly check ran")
					processRecurringMsg(recurringCommandMap[recurring.Hourly], dbPool, session)
				}
			case <-t.QuarterToHourly.C:
				if len(recurringCommandMap[recurring.QuarterToHourly]) != 0 {
					log.Println("Quarter-hourly check ran")
					processRecurringMsg(recurringCommandMap[recurring.HalfHourly], dbPool, session)
				}
			case <-t.HalfHourly.C:
				if len(recurringCommandMap[recurring.HalfHourly]) != 0 {
					log.Println("Half-hourly check ran")
					processRecurringMsg(recurringCommandMap[recurring.HalfHourly], dbPool, session)
				}
			case <-t.QuarterHourly.C:
				if len(recurringCommandMap[recurring.QuarterHourly]) != 0 {
					log.Println("Quarter-hourly check ran")
					processRecurringMsg(recurringCommandMap[recurring.HalfHourly], dbPool, session)
				}
			case <-t.TenMinutely.C:
				if len(recurringCommandMap[recurring.TenMinutely]) != 0 {
					log.Println("Quarter-hourly check ran")
					processRecurringMsg(recurringCommandMap[recurring.HalfHourly], dbPool, session)
				}
			case <-t.FiveMinutely.C:
				if len(recurringCommandMap[recurring.FiveMinutely]) != 0 {
					log.Println("Quarter-hourly check ran")
					processRecurringMsg(recurringCommandMap[recurring.HalfHourly], dbPool, session)
				}
			case <-t.Minutely.C:
				if len(recurringCommandMap[recurring.Minutely]) != 0 {
					log.Println("Minutely check ran")
					processRecurringMsg(recurringCommandMap[recurring.Minutely], dbPool, session)
				}
			}
		}
	}()
}

// StopAll timers so no more events are sent on their channels
func (t Timers) StopAll() {
	t.Daily.Stop()
	t.Hourly.Stop()
	t.HalfHourly.Stop()
	t.Minutely.Stop()
}

func processRecurringMsg(cmds []*RecurringCommand, dbPool *pgxpool.Pool, session *discordgo.Session) {
	for _, cmd := range cmds {
		pendingMsgs := (*cmd).Check(dbPool)
		for channel, msgs := range pendingMsgs {
			log.Printf("%s -> %#v", channel, msgs)
			for _, msg := range msgs {
				_, err := session.ChannelMessageSend(channel, msg)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}
