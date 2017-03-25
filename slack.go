package main

import (
	"fmt"
	"github.com/nlopes/slack"
)

type QuestionResponse struct {
	msg slack.Msg
	err error
}

type Slack struct {
	apiClient      *slack.Client
	imChannelCache map[string]string
}

func (s Slack) GetChannelMembers(channelId string) ([]string, error) {
	channel, err := s.apiClient.GetChannelInfo(channelId)
	if err != nil {
		return []string{}, fmt.Errorf("Error fetching channel info: %v", err)
	}
	return channel.Members, nil
}

func (s Slack) AskQuestion(member string, question string) chan QuestionResponse {
	respChan := make(chan QuestionResponse, 0)
	go func() {
		if err := s.SendMessage(member, question); err != nil {
			respChan <- QuestionResponse{err: err}
			return
		}
	}()
	return respChan
}

func (s Slack) GetChannelForMemberIm(member string) (*string, error) {
	if channelId, ok := s.imChannelCache[member]; ok {
		return &channelId, nil
	}
	_, _, channelId, err := s.apiClient.OpenIMChannel(member)
	if err != nil {
		return nil, fmt.Errorf("Error opening IM channel with user %s: %v", member, err)
	}
	s.imChannelCache[member] = channelId
	return &channelId, nil
}

func (s Slack) SendMessage(member string, msg string) error {
	channel, err := s.GetChannelForMemberIm(member)
	if err != nil {
		return fmt.Errorf("Error getting direct message channel for user %s: %v", member, err)
	}
	if _, _, err := s.apiClient.PostMessage(*channel, msg, slack.NewPostMessageParameters()); err != nil {
		return fmt.Errorf("Error sending message to user %s: %v", member, err)
	}
	return nil
}

func (s Slack) GetLatestDirectMessage(member string) (*slack.Msg, error) {
	channel, err := s.GetChannelForMemberIm(member)
	if err != nil {
		return nil, fmt.Errorf("Error getting direct message channel for user %s: %v", member, err)
	}
	parameters := slack.NewHistoryParameters()
	parameters.Latest = "1"
	history, err := s.apiClient.GetChannelHistory(*channel, parameters)
	if err != nil {
		return nil, fmt.Errorf("Error getting channel history for %s: %v", *channel, err)
	}
	return &history.Messages[0].Msg, nil
}
