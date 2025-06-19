package bybit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"tradingbot/internal/fatal"

	"github.com/ansel1/merry"
)

type ClientHTTP struct {
	*http.Client
	Scheme      string
	Host        string
	BybitKey    string
	BybitSecret string
}

func (client *ClientHTTP) GetServerTime(ctx context.Context) (output *ServerTime, err error) {
	target := url.URL{
		Scheme: client.Scheme,
		Host:   client.Host,
		Path:   "/v5/market/time",
	}
	request, err := http.NewRequest(http.MethodGet, target.String(), nil)
	fatal.OnError(err)
	response, err := client.Do(request)
	fatal.OnError(err)
	if response.StatusCode != http.StatusOK {
		err = merry.Errorf("unexpected status code (%v)", response.StatusCode).WithHTTPCode(response.StatusCode)
		return
	}
	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *ClientHTTP) GetWalletBalance(ctx context.Context, input GetWalletBalanceInput) (output *WalletBalance, err error) {
	target := url.URL{
		Scheme: client.Scheme,
		Host:   client.Host,
		Path:   "/v5/account/wallet-balance",
	}
	values := make(url.Values)
	values.Set("coin", string(input.Currency))
	values.Set("accountType", string(input.AccountType))
	target.RawQuery = values.Encode()
	request, err := http.NewRequest(http.MethodGet, target.String(), nil)
	fatal.OnError(err)
	timestamp := strconv.FormatInt(input.TimestampUnixMilli, 10)
	receiveWindow := "5000"
	signature := client.generateSignature(timestamp, receiveWindow, target.RawQuery)
	request.Header.Set("X-BAPI-SIGN-TYPE", strconv.Itoa(2))
	request.Header.Set("X-BAPI-SIGN", signature)
	request.Header.Set("X-BAPI-API-KEY", client.BybitKey)
	request.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	request.Header.Set("X-BAPI-RECV-WINDOW", receiveWindow)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	fatal.OnError(err)
	if response.StatusCode != http.StatusOK {
		err = merry.Errorf("unexpected status code (%v)", response.StatusCode).WithHTTPCode(response.StatusCode)
		return
	}
	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&output)
	return
}

func (client *ClientHTTP) generateSignature(timestamp string, receiveWindow string, input string) string {
	payload := timestamp + client.BybitKey + receiveWindow + input
	mac := hmac.New(sha256.New, []byte(client.BybitSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
