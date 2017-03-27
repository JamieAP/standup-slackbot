package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jawher/mow.cli"
	"github.com/nlopes/slack"
)

const (
	NAME = "standup-slackbot"
	DESC = "A slackbot for standups"
)

func main() {
	app := cli.App(NAME, DESC)
	slackToken := app.String(cli.StringOpt{
		Name:   "slack-token",
		Desc:   "Slack API token",
		EnvVar: "SLACK_TOKEN",
	})
	standupChannelName := app.String(cli.StringOpt{
		Name:   "standup-channel",
		Desc:   "The Slack channel to use for standups",
		EnvVar: "STANDUP_CHANNEL",
	})
	standupTime := app.String(cli.StringOpt{
		Name:   "standup-time",
		Desc:   "The time standup should start in 24hr 00:00 format",
		EnvVar: "STANDUP_TIME",
	})
	standupLengthMins := app.Int(cli.IntOpt{
		Name:   "standup-length-mins",
		Desc:   "The standup length time in minutes",
		EnvVar: "STANDUP_LENGTH_MINS",
		Value:  60,
	})
	timeZone := app.String(cli.StringOpt{
		Name:   "time-zone",
		Desc:   "The timezone IANA format e.g. Europe/London",
		EnvVar: "TIME_ZONE",
		Value:  "Europe/London",
	})
	app.Action = func() {
		var lastStandupDay *int = nil
		tz, err := time.LoadLocation(*timeZone)
		if err != nil {
			log.Fatalf("Error getting location for timezone: %v", err)
		}
		for {
			<-time.After(1 * time.Minute)
			now := time.Now().In(tz)

			isWeekend := now.Weekday() < 1 || now.Weekday() > 5
			if isWeekend {
				continue
			}

			day := now.Day()

			standupAlreadyDone := lastStandupDay != nil && *lastStandupDay == day
			if standupAlreadyDone {
				continue
			}

			hour, mins, err := parseStandupStartTime(standupTime)
			if err != nil {
				log.Fatalf("Error parsing standup start time: %v", err)
			}

			notTimeYet := now.Hour() < *hour || now.Minute() > *mins
			if notTimeYet {
				continue
			}

			standupStartTime := time.Date(now.Year(), now.Month(), now.Day(), *hour, *mins, 0, 0, tz)
			standupDuration := time.Minute * time.Duration(*standupLengthMins)

			// this prevents us doing standup in the case where we have no prior state (due to a restart)
			// but have already done standup for the day
			standupEndTimePassed := lastStandupDay == nil && standupStartTime.Add(standupDuration).Before(now)
			if standupEndTimePassed {
				continue
			}

			if err := DoStandup(*slackToken, *standupChannelName, *standupLengthMins); err != nil {
				log.Fatalf("Error doing standup: %v", err)
			}

			lastStandupDay = &day
		}
	}
	app.Run(os.Args)
}

func parseStandupStartTime(standupTime *string) (*int, *int, error) {
	hoursAndMins := strings.Split(*standupTime, ":")
	hour, err := strconv.ParseInt(hoursAndMins[0], 10, 8)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not parse hours from standup start time: %v", err)
	}
	mins, err := strconv.ParseInt(hoursAndMins[1], 10, 8)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not parse mins from standup start time: %v", err)
	}
	hourInt := int(hour)
	minsInt := int(mins)
	return &hourInt, &minsInt, nil
}

func DoStandup(slackToken string, standupChannelName string, standupLengthMins int) error {
	baseParams := slack.NewPostMessageParameters()
	baseParams.Username = "Standup Bot"

	slackClient := &Slack{
		slack.New(slackToken),
		make(map[string]string),
		make(map[string]func(event *slack.MessageEvent)),
		sync.Mutex{},
		baseParams,
	}

	channelId, err := slackClient.GetChannelIdForChannel(standupChannelName)
	if err != nil {
		return fmt.Errorf("Could not get channel ID for channel %s: %v", standupChannelName, err)
	}

	members, err := slackClient.GetChannelMembers(*channelId)
	if err != nil {
		return fmt.Errorf("Error getting standup channel members: %v", err)
	}

	standup := NewStandup(slackClient, time.Now().Add(time.Minute*time.Duration(standupLengthMins)), members)
	results := standup.Start()

	for i := 0; i < 5; i++ {
		_, _, err = slackClient.apiClient.PostMessage(
			*channelId,
			"Standup is finished, keep up the good work team!",
			BuildSlackReport(baseParams, results),
		)
		if err == nil {
			break
		}
		log.Printf("Error posting standup result to Slack: %v", err)
		<-time.After(time.Duration(3*math.Pow(2, float64(i+1))) * time.Second)
	}
	return nil
}

func BuildSlackReport(baseParams slack.PostMessageParameters, questionnaires map[string]*StandupQuestionnaire) slack.PostMessageParameters {
	postParams := baseParams
	attachments := make([]slack.Attachment, len(questionnaires))
	for _, questionnaire := range questionnaires {
		attachment := slack.Attachment{
			Fallback:   "There is no fallback text...",
			AuthorIcon: questionnaire.Member.Profile.Image48,
			AuthorName: questionnaire.Member.RealName,
			Fields: []slack.AttachmentField{
				{
					Title: "What did you get done since last standup?",
					Short: false,
					Value: questionnaire.yesterday,
				},
				{
					Title: "What are you working on today?",
					Short: false,
					Value: questionnaire.today,
				},
				{
					Title: "When do you think you'll be finished?",
					Short: false,
					Value: questionnaire.finishedWhen,
				},
				{
					Title: "Is there anything blocking you?",
					Short: false,
					Value: questionnaire.blockers,
				},
			},
		}
		attachments = append(attachments, attachment)
	}
	postParams.Attachments = attachments
	return postParams
}
