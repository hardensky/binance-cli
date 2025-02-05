package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/adshao/go-binance"
	"github.com/juju/errors"
	"gopkg.in/urfave/cli.v1"
)

var (
	name     string
	keyfile  string
	debug    bool
	accounts map[string]*Account
	assets   []string
)

// AccountKey define key info for account
type AccountKey struct {
	Name      string `json:"name"`
	APIKey    string `json:"api_key"`
	SecretKey string `json:"secret_key"`
}

func loadKeys(filePath string) ([]AccountKey, error) {
	keyBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var keys []AccountKey
	err = json.Unmarshal(keyBytes, &keys)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return keys, nil
}

func initAccounts() {
	if keyfile == "" {
		keyfile = "keys.json"
	}
	keys, err := loadKeys(keyfile)
	if err != nil {
		log.Fatal("failed to load keys: ", err)
	}
	accounts = make(map[string]*Account)
	for _, key := range keys {
		client := binance.NewClient(
			key.APIKey,
			key.SecretKey,
		)
		if debug {
			client.Debug = true
		}
		account := new(Account)
		account.Client = client
		account.Name = key.Name
		accounts[account.Name] = account
	}
}

var findAccounts func(name string) map[string]*Account

func findAccountsImpl(name string) map[string]*Account {
	initAccounts()
	if name == "" {
		return accounts
	}
	return map[string]*Account{name: accounts[name]}
}

func runOnce(action func(*Account) (interface{}, error),
	postAction ...func(map[string]interface{}) (interface{}, error)) error {
	findAccounts = func(name string) map[string]*Account {
		initAccounts()
		for k, v := range accounts {
			return map[string]*Account{k: v}
		}
		return nil
	}
	return accountsDo(action, postAction...)
}

func accountsDo(action func(*Account) (interface{}, error),
	postAction ...func(map[string]interface{}) (interface{}, error)) error {
	accounts := findAccounts(name)
	var ret interface{}
	var err error
	results := make(map[string]interface{})
	for _, account := range accounts {
		res, err := action(account)
		if err != nil {
			// return errors.Trace(err)
			results[account.Name] = fmt.Sprintf("error: %s", err)
		} else {
			results[account.Name] = res
		}
	}
	if len(postAction) > 0 {
		ret, err = postAction[0](results)
		if err != nil {
			return errors.Trace(err)
		}
	} else {
		ret = results
	}
	return print(ret)
}

func print(ret interface{}) error {
	out, err := json.MarshalIndent(ret, "", "    ")
	if err != nil {
		return errors.Trace(err)
	}
	fmt.Println(string(out))
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "binance-cli"
	app.Usage = "Binance CLI"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "name",
			Usage:       "account name",
			Destination: &name,
		},
		cli.StringFlag{
			Name:        "keyfile",
			Usage:       "file path of api keys",
			Destination: &keyfile,
		},
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "show debug info",
			Destination: &debug,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "list-balances",
			Usage: "list account balances",
			Flags: []cli.Flag{
				cli.StringSliceFlag{
					Name:   "assets",
					EnvVar: "BINANCE_ASSETS",
					Usage:  "list balances with asset BTC, BNB ...",
					Value:  &cli.StringSlice{"BTC", "BNB", "WINK", "USDT"},
				},
				cli.BoolTFlag{
					Name:  "total",
					Usage: "show total balance",
				},
			},
			Action: func(c *cli.Context) error {
				return listBalances(c.StringSlice("assets"), c.Bool("total"))
			},
		},
		{
			Name:  "list-prices",
			Usage: "list latest price for a symbol or symbols",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "symbol",
					Usage: "filter with symbol",
				},
			},
			Action: func(c *cli.Context) error {
				return listPrices(c.String("symbol"))
			},
		},
		{
			Name:  "list-orders",
			Usage: "list open orders",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "symbol",
					Usage: "list orders with symbol",
				},
			},
			Action: func(c *cli.Context) error {
				return listOpenOrders(c.String("symbol"))
			},
		},
		{
			Name:  "create-order",
			Usage: "create order",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "symbol",
					Usage: "symbol name: BNBBTC",
				},
				cli.StringFlag{
					Name:  "side",
					Usage: "side type: SELL or BUY",
				},
				cli.StringFlag{
					Name:  "quantity",
					Usage: "quantity of symbol",
				},
				cli.StringFlag{
					Name:  "price",
					Usage: "price of symbol",
				},
			},
			Action: func(c *cli.Context) error {
				return createOrder(
					c.String("symbol"), c.String("side"),
					c.String("quantity"), c.String("price"))
			},
		},
		{
			Name:  "cancel-orders",
			Usage: "cancel open orders",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "symbol",
					Usage: "cancel open orders with symbol",
				},
			},
			Action: func(c *cli.Context) error {
				return cancelOrders(c.String("symbol"))
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(errors.ErrorStack(err))
	}
}

func init() {
	findAccounts = findAccountsImpl
}
