package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/broker/tastytrade"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/manifoldco/promptui"
)

func main() {
	ctx := context.Background()
	cli := NewTastyTradeCLI()
	err := cli.SetAccountID(ctx)
	fatal.OnError(err)
	err = cli.SetSymbol(ctx)
	fatal.OnError(err)

	fmt.Printf("Selected account: %s\n", cli.accountID)
	fmt.Printf("Selected symbol: %s\n", cli.symbol)

	err = cli.StartStream(ctx)
	fatal.OnError(err)
}

type TastyTradeCLI struct {
	accountID string
	symbol    string
	client    tastytrade.Client
}

func NewTastyTradeCLI() *TastyTradeCLI {
	credentialsByType := auth.CredentialsByTypeFromEnv()
	credentials, ok := credentialsByType["tastytrade"]
	fatal.Unless(ok)
	apiURL, err := url.Parse(credentials.APIURL)
	fatal.OnError(err)
	tokenManager := auth.NewTastyTradeTokenManager(&credentials.AuthorizationServer)
	tastyTradeClient := tastytrade.NewHTTPClient(tastytrade.NewHTTPClientInput{
		APIURL:         apiURL,
		GetAccessToken: tokenManager.GetAccessToken,
	})
	return &TastyTradeCLI{
		client: tastyTradeClient,
	}
}

func (cli *TastyTradeCLI) SetAccountID(ctx context.Context) (err error) {
	accounts, err := cli.client.ListAccounts(ctx)
	if err != nil {
		return
	}
	var accountIDs []string
	for _, account := range accounts {
		for _, item := range account.Data.Items {
			accountIDs = append(accountIDs, item.Account.AccountNumber)
		}
	}
	selectAccountPrompt := promptui.Select{
		Label: "Select account ID",
		Items: accountIDs,
	}
	_, cli.accountID, err = selectAccountPrompt.Run()
	return
}

// BTC/USD:CXTALP
func (cli *TastyTradeCLI) SetSymbol(ctx context.Context) (err error) {
	symbolPrompt := promptui.Prompt{
		Label: "Enter symbol",
		Validate: func(symbol string) error {
			// symbol = strings.ToUpper(strings.TrimSpace(symbol))
			// _, err := cli.client.SearchSymbol(ctx, symbol)
			// if err != nil {
			// 	return err
			// }
			cli.symbol = symbol
			return nil
		},
	}
	_, err = symbolPrompt.Run()
	return
}

func (cli *TastyTradeCLI) StartStream(ctx context.Context) (err error) {
	iterator := tastytrade.NewDXLinkIterator(ctx, tastytrade.NewDXLinkIteratorInput{
		Client: cli.client,
		Symbol: cli.symbol,
	})
	fmt.Printf("Subscribed to %s. Streaming market data...\n", cli.symbol)
	for iterator.Next() {
		fmt.Printf("%+v\n", iterator.Message())
	}
	return iterator.Err()
}
