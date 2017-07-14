package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

type QuestionResponse struct {
	msg slack.Msg
	err error
}

type Slack struct {
	apiClient                *slack.Client
	imChannelCache           map[string]string
	messageEventHandlers     map[string]func(event *slack.MessageEvent)
	messageEventHandlersLock sync.Mutex
	BaseMessageParams        slack.PostMessageParameters
}

func (s *Slack) StartRealTimeMessagingListener(ctx context.Context, standupIsOver *bool) {
	rtm, s := connect(s, ctx)
	for msg := range rtm.IncomingEvents {
		switch event := msg.Data.(type) {
		case *slack.MessageEvent:
			go func() {
				for _, handler := range s.messageEventHandlers {
					go handler(event)
				}
			}()
		case *slack.RTMError:
			log.Printf("Error received on RTM channel: %v", event.Error())
		case *slack.DisconnectedEvent:
			if !*standupIsOver {
				log.Printf("Reconnecting to RTM channel because standup is not over yet, standupIsOver: %+v.\n", *standupIsOver)
				rtm, _ = connect(s, ctx)
			} else {
				log.Printf("Disconnected from RTM channel, completed: %+v", *standupIsOver)
				return
			}
		}
	}
}
func connect(s *Slack, ctx context.Context) (*slack.RTM, *Slack) {
	rtm := s.apiClient.NewRTM()
	go rtm.ManageConnection()
	go func(rtm *slack.RTM) {
		select {
		case <-ctx.Done():
			if err := rtm.Disconnect(); err != nil {
				log.Printf("Error disconnecting from RTM channel: %v", err)
			}
		}
	}(rtm)
	return rtm, s
}

// returns a uuid identifying the event handler that can be used with RemoveMessageEventHandler
func (s *Slack) AddMessageEventHandler(handler func(event *slack.MessageEvent)) string {
	s.messageEventHandlersLock.Lock()
	defer s.messageEventHandlersLock.Unlock()
	uuid := uuid.NewV4().String()
	s.messageEventHandlers[uuid] = handler
	return uuid
}

func (s *Slack) RemoveMessageEventHandler(uuid string) {
	s.messageEventHandlersLock.Lock()
	defer s.messageEventHandlersLock.Unlock()
	delete(s.messageEventHandlers, uuid)
}

func (s *Slack) GetChannelMembers(channelId string) (map[string]*slack.User, error) {
	members := make(map[string]*slack.User)
	channel, err := s.apiClient.GetChannelInfo(channelId)
	if err != nil {
		return members, fmt.Errorf("Error fetching channel info: %v", err)
	}
	for _, member := range channel.Members {
		memberInfo, err := s.apiClient.GetUserInfo(member)
		if err != nil {
			return members, fmt.Errorf("Error getting member info for %s: %v", member, err)
		}
		members[member] = memberInfo
	}

	return members, nil
}

func (s *Slack) AskQuestion(member string, question string, ctx context.Context) QuestionResponse {
	respChan := make(chan QuestionResponse, 0)
	questionTs, err := s.SendMessage(member, question)
	if err != nil {
		return QuestionResponse{err: err}
	}
	handlerUuid := s.AddMessageEventHandler(func(event *slack.MessageEvent) {
		eventTs, err := parseTimestamp(event.Timestamp)
		if err != nil {
			respChan <- QuestionResponse{err: err}
			return
		}
		channel, err := s.GetChannelForMemberIm(member)
		if err != nil {
			respChan <- QuestionResponse{err: err}
		}
		// due to Slack's timestamp precision, it's possible for time.After to return false if the messages are
		// sent close together enough, thus !time.Before is used.
		if event.Channel == *channel && event.User == member && !eventTs.Before(*questionTs) {
			respChan <- QuestionResponse{msg: event.Msg}
		}
	})
	select {
	case resp := <-respChan:
		s.RemoveMessageEventHandler(handlerUuid)
		return resp
	case <-ctx.Done():
		s.RemoveMessageEventHandler(handlerUuid)
		return QuestionResponse{err: fmt.Errorf("Context cancelled: %v", ctx.Err())}
	}
}

func (s *Slack) GetChannelForMemberIm(member string) (*string, error) {
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

func (s *Slack) GetChannelIdForChannel(channelName string) (*string, error) {
	channels, err := s.apiClient.GetChannels(true)
	if err != nil {
		return nil, fmt.Errorf("Error getting channel list: %v", err)
	}
	for _, channel := range channels {
		if channel.Name == channelName {
			return &channel.ID, nil
		}
	}
	return nil, errors.New("Channel not found")
}

func (s *Slack) SendMessage(member string, msg string) (*time.Time, error) {
	channel, err := s.GetChannelForMemberIm(member)
	if err != nil {
		return nil, fmt.Errorf("Error getting direct message channel for user %s: %v", member, err)
	}
	_, ts, err := s.apiClient.PostMessage(*channel, msg, s.BaseMessageParams)
	if err != nil {
		return nil, fmt.Errorf("Error sending message to user %s: %v", member, err)
	}
	return parseTimestamp(ts)
}

func parseTimestamp(timestamp string) (*time.Time, error) {
	tsParts := strings.Split(timestamp, ".")
	if len(tsParts) != 2 {
		return nil, fmt.Errorf("Malformed unix timestamp: %s", timestamp)
	}
	secs, err := strconv.ParseInt(tsParts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Malformed unix timestamp: %s", timestamp)
	}
	nanos, err := strconv.ParseInt(tsParts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Malformed unix timestamp: %s", timestamp)
	}
	ts := time.Unix(secs, nanos)
	return &ts, nil
}
