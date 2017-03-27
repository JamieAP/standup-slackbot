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
}

func (s *Slack) StartRealTimeMessagingListener(ctx context.Context) {
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
			log.Print("Disconnected from RTM channel")
			return
		}
	}
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
	questionSentAt := time.Now()
	if err := s.SendMessage(member, question); err != nil {
		return QuestionResponse{err: err}
	}
	channel, err := s.GetChannelForMemberIm(member)
	if err != nil {
		return QuestionResponse{err: err}
	}
	handlerUuid := s.AddMessageEventHandler(func(event *slack.MessageEvent) {
		timestamp, err := parseTimestamp(event.Timestamp)
		if err != nil {
			respChan <- QuestionResponse{err: err}
			return
		}
		if event.Channel == *channel && event.User == member && timestamp.After(questionSentAt) {
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

func (s *Slack) SendMessage(member string, msg string) error {
	channel, err := s.GetChannelForMemberIm(member)
	if err != nil {
		return fmt.Errorf("Error getting direct message channel for user %s: %v", member, err)
	}
	if _, _, err := s.apiClient.PostMessage(*channel, msg, slack.NewPostMessageParameters()); err != nil {
		return fmt.Errorf("Error sending message to user %s: %v", member, err)
	}
	return nil
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
