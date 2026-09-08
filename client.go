package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/seabird-chat/seabird-go"
	"github.com/seabird-chat/seabird-go/pb"
)

// SeabirdClient is a basic client for seabird
type SeabirdClient struct {
	context.Context
	*seabird.Client
	market  marketData
	command string
	logger  zerolog.Logger
}

const defaultCommand = "stock"

// NewSeabirdClient returns a new seabird client
func NewSeabirdClient(seabirdCoreURL, seabirdCoreToken, finnhubToken, command string, logger zerolog.Logger) (*SeabirdClient, error) {
	seabirdClient, err := seabird.NewClient(seabirdCoreURL, seabirdCoreToken)
	if err != nil {
		return nil, err
	}
	if command == "" {
		command = defaultCommand
	}

	return &SeabirdClient{
		Context: context.Background(),
		Client:  seabirdClient,
		market:  newFinnhubMarket(finnhubToken),
		command: command,
		logger:  logger,
	}, nil
}

func (c *SeabirdClient) handles(command string) bool {
	switch command {
	case c.command, c.command + "s":
		return true
	case "stonk", "stonks":
		return c.command == defaultCommand
	}
	return false
}

func (c *SeabirdClient) close() error {
	return c.Client.Close()
}

func (c *SeabirdClient) reply(source *pb.ChannelSource, format string, args ...interface{}) {
	if err := c.MentionReplyf(source, format, args...); err != nil {
		c.logger.Error().Err(err).Str("channel_id", source.GetChannelId()).Msg("failed to send reply")
	}
}

func (c *SeabirdClient) stockCallback(event *pb.CommandEvent) {
	arg := strings.TrimSpace(event.Arg)

	cmdLog := c.logger.With().
		Str("command", event.Command).
		Str("arg", arg).
		Str("channel_id", event.Source.GetChannelId()).
		Logger()

	text, err := respond(c.Context, c.market, event.Command, arg, time.Now())
	if err != nil {
		cmdLog.Error().Err(err).Msg("finnhub lookup failed")
		c.reply(event.Source, "Unable to reach Finnhub right now.")
		return
	}
	cmdLog.Info().Str("reply", text).Msg("handled command")
	c.reply(event.Source, "%s", text)
}

// Run runs
func (c *SeabirdClient) Run() error {
	events, err := c.StreamEvents(map[string]*pb.CommandMetadata{
		c.command: {
			Name:      c.command,
			ShortHelp: usage(c.command),
			FullHelp:  "Stock quotes, returns, fundamentals, earnings, analyst ratings, news, and market status from Finnhub",
		},
	})
	if err != nil {
		return err
	}

	c.logger.Info().Str("command", c.command).Msg("event stream open")

	for event := range events.C {
		switch v := event.GetInner().(type) {
		case *pb.Event_Command:
			if c.handles(v.Command.Command) {
				go c.stockCallback(v.Command)
			}
		}
	}

	if closeErr := events.Close(); closeErr != nil {
		return fmt.Errorf("event stream closed: %w", closeErr)
	}
	return errors.New("event stream closed without error")
}
