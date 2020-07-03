package stock

import (
	"context"
	"fmt"
	"time"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go"
	"github.com/antihax/optional"
	"github.com/jaredledvina/seabird-stock-plugin/pb"
	"google.golang.org/grpc"
)

// SeabirdClient is a basic client for seabird
type SeabirdClient struct {
	grpcChannel  *grpc.ClientConn
	inner        pb.SeabirdClient
	FinnhubToken string
}

// NewSeabirdClient returns a new seabird client
func NewSeabirdClient(seabirdCoreURL, seabirdCoreToken, finnhubToken string) (*SeabirdClient, error) {
	grpcChannel, err := NewGRPCClient(seabirdCoreURL, seabirdCoreToken)
	if err != nil {
		return nil, err
	}

	return &SeabirdClient{
		grpcChannel:  grpcChannel,
		inner:        pb.NewSeabirdClient(grpcChannel),
		FinnhubToken: finnhubToken,
	}, nil
}

func (c *SeabirdClient) close() error {
	return c.grpcChannel.Close()
}

func (c *SeabirdClient) reply(source *pb.ChannelSource, msg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.inner.SendMessage(ctx, &pb.SendMessageRequest{
		ChannelId: source.GetChannelId(),
		Text:      msg,
	})
	return err
}

func (c *SeabirdClient) stockCallback(event *pb.CommandEvent) {
	go func() {

		// TODO: Request debugging
		fmt.Printf("Processing event: %s %s %s", event.Source, event.Command, event.Arg)

		finnhubClient := finnhub.NewAPIClient(finnhub.NewConfiguration()).DefaultApi
		auth := context.WithValue(context.Background(), finnhub.ContextAPIKey, finnhub.APIKey{
			Key: c.FinnhubToken,
		})
		ticker := event.Arg
		quote, _, err := finnhubClient.Quote(auth, ticker)
		if err != nil {
			// TODO: What do we do with the error?
			fmt.Println(err)
		}
		profile2, _, err := finnhubClient.CompanyProfile2(auth, &finnhub.CompanyProfile2Opts{Symbol: optional.NewString(ticker)})
		if err != nil {
			// TODO: What do we do with the error?
			fmt.Println(err)
		}
		var company string
		if profile2.Name != "" {
			company = fmt.Sprintf("%s (%s)", profile2.Name, ticker)
		} else {
			company = fmt.Sprintf("%s", ticker)
		}
		// TODO: Don't hardcoded USD here
		msg := fmt.Sprintf("%s: Current price of %s is: $%+v USD", event.Source.GetUser().GetDisplayName(), company, quote.C)
		c.reply(event.Source, msg)
	}()
}

// Run runs
func (c *SeabirdClient) Run() error {
	events, err := c.inner.StreamEvents(
		context.Background(),
		&pb.StreamEventsRequest{
			Commands: map[string]*pb.CommandMetadata{
				"stock": {
					Name:      "stock",
					ShortHelp: "<ticker>",
					FullHelp:  "Returns current stock price for given ticker",
				},
			},
		},
	)
	if err != nil {
		return err
	}

	for {
		event, err := events.Recv()
		if err != nil {
			return err
		}
		switch v := event.GetInner().(type) {
		case *pb.Event_Command:
			if v.Command.Command == "stock" {
				c.stockCallback(v.Command)
			}
		}
	}
}
