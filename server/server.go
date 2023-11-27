package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"sync"

	_ "github.com/davecgh/go-spew/spew"
	"github.com/djeday123/crypto-exchange/orderbook"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/gookit/goutil/dump"
	"github.com/labstack/echo/v4"
)

const (
	MarketETH Market = "ETH"

	MarketOrder OrderType = "MARKET"
	LimitOrder  OrderType = "LIMIT"

	exchangePrivateKey = "e2dff7b2e7ba5d5a7cfc8cb00c44a0e606c8e0b57e29460092276968f4e527ee"
)

type (
	OrderType string
	Market    string

	PlaceOrderRequest struct {
		UserID int64
		Type   OrderType // Limit or market
		Bid    bool
		Size   float64
		Price  float64
		Market Market
	}

	Order struct {
		UserID    int64
		ID        int64
		Price     float64
		Size      float64
		Bid       bool
		Timestamp int64
	}

	OrderbookData struct {
		TotalBidVolume float64
		TotalAskVolume float64
		Asks           []*Order
		Bids           []*Order
	}

	MatchedOrders struct {
		UserID int64
		Price  float64
		Size   float64
		ID     int64
	}

	APIError struct {
		Error string
	}
)

func StartServer() {
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler

	client, err := ethclient.Dial("http://127.0.0.1:7545")
	if err != nil {
		log.Fatal(err)
	}

	ex, err := NewExchange(exchangePrivateKey, client)
	if err != nil {
		log.Fatal(err)
	}

	pkStr8 := "5c800ab69e3acf17f166a2f414cd9113d896883b342bedec561141d136a58cfb"
	user8 := NewUser(pkStr8, 8)
	ex.Users[user8.ID] = user8

	buyerAddressStr := "0x8573C870D29E1bD5f60aAa437c990DeD07910f14"
	buyerBalance, err := client.BalanceAt(context.Background(), common.HexToAddress(buyerAddressStr), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("buyer: ", buyerBalance)

	//----------------------------------------------------------------------

	pkStr7 := "8783bdd2ca8c83e67f1a60f1b6c9e028d26445634ca6ef2f5168c2edafd39d0c"
	user7 := NewUser(pkStr7, 7)
	ex.Users[user7.ID] = user7

	sellerAddressStr := "0x6A51d3e3904EC92e5634114983E9b9B0C6dFa2f5"
	sellerBalance, err := client.BalanceAt(context.Background(), common.HexToAddress(sellerAddressStr), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("seller: ", sellerBalance)

	//----------------------------------------------------------------------

	johnPk := "e41b654b440582ab6e0f3739f29cfaed5f17571fd0ddac5faccf30784927d92d"
	john := NewUser(johnPk, 6)
	ex.Users[john.ID] = john

	johnAddressStr := "0x04B18089Ac15d317bED921834d51170Acc6dFBFF"
	johnBalance, err := client.BalanceAt(context.Background(), common.HexToAddress(johnAddressStr), nil)
	if err != nil {
		log.Fatal(err)
	}

	privateKey2, err := crypto.HexToECDSA(johnPk)
	johnPublicKeyECDSA, ok := privateKey2.Public().(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	johnAddressStr2 := crypto.PubkeyToAddress(*johnPublicKeyECDSA)

	fmt.Println("john address: ", johnAddressStr2)
	fmt.Println("john address: ", johnAddressStr)
	fmt.Println("john: ", johnBalance)

	//----------------------------------------------------------------------

	e.POST("/order", ex.handlePlaceOrder)

	e.GET("/trades/:market", ex.handleGetTrades)
	e.GET("/order/:userID", ex.handleGetOrders)
	e.GET("/book/:market", ex.handleGetBook)
	e.GET("/book/:market/bid", ex.handleGetBestBid)
	e.GET("/book/:market/ask", ex.handleGetBestAsk)

	e.DELETE("/order/:id", ex.CancelOrder)

	//--------------------------------

	e.Start(":3010")
}

type User struct {
	ID         int64
	PrivateKey *ecdsa.PrivateKey
}

func NewUser(privKey string, id int64) *User {
	pk, err := crypto.HexToECDSA(privKey)
	if err != nil {
		panic(err)
	}

	return &User{
		ID:         id,
		PrivateKey: pk,
	}
}

func httpErrorHandler(err error, c echo.Context) {
	fmt.Println(err)
}

type Exchange struct {
	Client     *ethclient.Client
	mu         sync.RWMutex
	Users      map[int64]*User
	Orders     map[int64][]*orderbook.Order
	PrivateKey *ecdsa.PrivateKey
	orderbooks map[Market]*orderbook.Orderbook
}

//Orders maps user to his orders
// orders     map[int64]int64

func NewExchange(privateKey string, client *ethclient.Client) (*Exchange, error) {
	orderbooks := make(map[Market]*orderbook.Orderbook)
	orderbooks[MarketETH] = orderbook.NewOrderbook()

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}
	return &Exchange{
		Client: client,
		Users:  make(map[int64]*User),
		//orders:     make(map[int64]int64),
		Orders:     make(map[int64][]*orderbook.Order),
		PrivateKey: pk,
		orderbooks: orderbooks,
	}, nil
}

type GetOrdersResponse struct {
	Asks []Order
	Bids []Order
}

func (ex *Exchange) handleGetTrades(c echo.Context) error {
	market := Market(c.Param("market"))

	ob, ok := ex.orderbooks[market]
	if !ok {
		return c.JSON(http.StatusBadGateway, APIError{Error: "orderbook not found!"})
	}
	return c.JSON(http.StatusOK, ob.Trades)
}

func (ex *Exchange) handleGetOrders(c echo.Context) error {
	userIDSTR := c.Param("userID")
	userID, err := strconv.Atoi(userIDSTR)
	if err != nil {
		return err
	}

	ex.mu.RLock()
	orderbookOrders := ex.Orders[int64(userID)]
	ordersResp := &GetOrdersResponse{
		Asks: []Order{},
		Bids: []Order{},
	}

	//orders := make([]Order, len(orderbookOrders))
	//fmt.Printf("%v \n %p \n %v\n", &y, y, *y)
	for i := 0; i < len(orderbookOrders); i++ {

		if orderbookOrders[i].Limit == nil {
			fmt.Printf("the limit of the order is NIL %+v\n", orderbookOrders[i])
			continue
		}

		order := Order{
			ID:        orderbookOrders[i].ID,
			UserID:    orderbookOrders[i].UserID,
			Price:     orderbookOrders[i].Limit.Price,
			Size:      orderbookOrders[i].Size,
			Timestamp: orderbookOrders[i].Timestamp,
			Bid:       orderbookOrders[i].Bid,
		}

		if order.Bid {
			ordersResp.Bids = append(ordersResp.Bids, order)
		} else {
			ordersResp.Asks = append(ordersResp.Asks, order)
		}
	}
	ex.mu.RUnlock()

	return c.JSON(http.StatusOK, ordersResp)
}

func (ex *Exchange) handleGetBook(c echo.Context) error {
	market := Market(c.Param("market"))
	ob, ok := ex.orderbooks[market]

	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{"msg": "market not found"})
	}

	orderbookData := OrderbookData{
		TotalBidVolume: ob.BidTotalVolume(),
		TotalAskVolume: ob.AskTotalVolume(),
		Asks:           []*Order{},
		Bids:           []*Order{},
	}

	for _, limit := range ob.Asks() {
		for _, order := range limit.Orders {
			o := Order{
				UserID:    order.UserID,
				ID:        order.ID,
				Price:     limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Asks = append(orderbookData.Asks, &o)
		}
	}

	for _, limit := range ob.Bids() {
		for _, order := range limit.Orders {
			o := Order{
				UserID:    order.UserID,
				ID:        order.ID,
				Price:     limit.Price,
				Size:      order.Size,
				Bid:       order.Bid,
				Timestamp: order.Timestamp,
			}
			orderbookData.Bids = append(orderbookData.Bids, &o)
		}
	}

	return c.JSON(http.StatusOK, orderbookData)
}

type PriceResponse struct {
	Price float64
}

func (ex *Exchange) handleGetBestBid(c echo.Context) error {
	market := Market(c.Param("market"))
	ob := ex.orderbooks[market]

	if len(ob.Bids()) == 0 {
		return fmt.Errorf("the bids are empty")
	}

	bestBidPrice := ob.Bids()[0].Price
	pr := PriceResponse{
		Price: bestBidPrice,
	}

	return c.JSON(http.StatusOK, pr)
}

func (ex *Exchange) handleGetBestAsk(c echo.Context) error {
	market := Market(c.Param("market"))
	ob := ex.orderbooks[market]

	if len(ob.Asks()) == 0 {
		return fmt.Errorf("the asks are empty")
	}

	bestAskPrice := ob.Asks()[0].Price
	pr := PriceResponse{
		Price: bestAskPrice,
	}

	return c.JSON(http.StatusOK, pr)
}

func (ex *Exchange) CancelOrder(c echo.Context) error {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	ob := ex.orderbooks[MarketETH]
	order := ob.Orders[int64(id)]
	ob.CancelOrder(order)

	log.Println("order canceled id => ", id)

	return c.JSON(200, map[string]any{"msg": "order deleted"})
}

func (ex *Exchange) handlePlaceMarketOrder(market Market, order *orderbook.Order) ([]orderbook.Match, []*MatchedOrders) {
	ob := ex.orderbooks[market]

	ex.mu.Lock()
	matches := ob.PlaceMarketOrder(order)
	matchedOrders := make([]*MatchedOrders, len(matches))

	isBid := false
	if order.Bid {
		isBid = true
	}

	totalSizeFilled := 0.0
	sumPrice := 0.0
	for i := 0; i < len(matchedOrders); i++ {
		id := matches[i].Bid.ID
		limitUserID := matches[i].Bid.UserID
		if isBid {
			id = matches[i].Ask.ID
			limitUserID = matches[i].Ask.UserID
		}
		matchedOrders[i] = &MatchedOrders{
			UserID: limitUserID,
			ID:     id,
			Size:   matches[i].SizeFilled,
			Price:  matches[i].Price,
		}
		totalSizeFilled += matches[i].SizeFilled
		sumPrice += matches[i].Price
	}

	avgPrice := sumPrice / float64(len(matches))

	log.Printf("filled market order => %d | size: [%.2f] | avgPrice: [%.2f]", order.ID, totalSizeFilled, avgPrice)
	newOrderMap := make(map[int64][]*orderbook.Order)

	for userID, orderbookOrders := range ex.Orders {
		for i := 0; i < len(orderbookOrders); i++ {
			//If the order is not filled we place it in the map copy.
			// this means that size of the order = 0
			if !orderbookOrders[i].IsFilled() {
				newOrderMap[userID] = append(newOrderMap[userID], orderbookOrders[i])
			}
		}
	}
	ex.Orders = newOrderMap
	ex.mu.Unlock()

	return matches, matchedOrders
}

func (ex *Exchange) handlePlaceLimitOrder(market Market, price float64, order *orderbook.Order) error {
	ob := ex.orderbooks[market]
	ob.PlaceLimitOrder(price, order)

	//keep track of the orders
	ex.mu.Lock()
	ex.Orders[order.UserID] = append(ex.Orders[order.UserID], order)
	ex.mu.Unlock()

	log.Printf("new LIMIT order => ID: [%d] | userID: [%d] | type: [%t] | price [%.2f] | size [%.2f]", order.ID, order.UserID, order.Bid, order.Limit.Price, order.Size)

	return nil
}

type PlaceOrderResponse struct {
	OrderID int64
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderData PlaceOrderRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderData); err != nil {
		return err
	}

	market := Market(placeOrderData.Market)
	order := orderbook.NewOrder(placeOrderData.Bid, placeOrderData.Size, placeOrderData.UserID)

	// Limit orders
	if placeOrderData.Type == LimitOrder {
		if err := ex.handlePlaceLimitOrder(market, placeOrderData.Price, order); err != nil {
			return err
		}
	}

	// market orders
	if placeOrderData.Type == MarketOrder {
		matches, _ := ex.handlePlaceMarketOrder(market, order)

		if err := ex.handleMatches(matches); err != nil {
			return err
		}

	}

	resp := &PlaceOrderResponse{
		OrderID: order.ID,
	}
	return c.JSON(200, resp)
}

func (ex *Exchange) handleMatches(matches []orderbook.Match) error {
	for _, match := range matches {
		fromUser, ok := ex.Users[match.Ask.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Ask.UserID)
		}
		//fromAddress := crypto.PubkeyToAddress(fromUser.PrivateKey.PublicKey)

		toUser, ok := ex.Users[match.Bid.UserID]
		if !ok {
			return fmt.Errorf("user not found: %d", match.Bid.UserID)
		}
		toAddress := crypto.PubkeyToAddress(toUser.PrivateKey.PublicKey)

		amount := big.NewInt(int64(match.SizeFilled))
		transferETH(ex.Client, fromUser.PrivateKey, toAddress, amount)
	}
	return nil
}

func transferETH(client *ethclient.Client, fromPrivKey *ecdsa.PrivateKey, to common.Address, amount *big.Int) error {
	ctx := context.Background()
	publicKey := fromPrivKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return err
	}

	gasLimit := uint64(21000) // in units
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatal(err)
	}

	tx := types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, nil)

	chainID := big.NewInt(1337)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), fromPrivKey)
	if err != nil {
		return err
	}

	return client.SendTransaction(ctx, signedTx)
}
