package main

import (
	"log"
	"sync"
	"time"

	"github.com/nlopes/slack"
)

func main() {
	channel := "C4PUNBEMQ"

	slackClient := &Slack{
		slack.New("xoxb-152520612096-sPKLUWO7FEYg0cMmPofGUyWt"),
		make(map[string]string),
		make(map[string]func(event *slack.MessageEvent)),
		sync.Mutex{},
	}

	members, err := slackClient.GetChannelMembers(channel)
	if err != nil {
		log.Fatalf("Error getting standup channel members: %v", err)
	}

	standup := NewStandup(slackClient, time.Now().Add(time.Hour*1), members)
	results := standup.Start()

	_, _, err = slackClient.apiClient.PostMessage(
		channel,
		"Standup is finished, keep up the good work team!",
		BuildSlackReport(results),
	)

	if err != nil {
		log.Printf("Error posting standup result to Slack: %v", err)
		log.Printf("Standup results: %+v", results)
	}
}

func BuildSlackReport(questionnaires map[string]*StandupQuestionnaire) slack.PostMessageParameters {
	postParams := slack.NewPostMessageParameters()
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
