package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/djeday123/crypto-exchange/client"
	"github.com/djeday123/crypto-exchange/server"
)

const (
	maxOrders = 3
	userID    = 7
)

var (
	tick = 2 * time.Second
)

func marketOrderPlacer(c *client.Client) {
	ticker := time.NewTicker(5 * time.Second)
	for {

		trades, err := c.GetTrades("ETH")
		if err != nil {
			panic(err)
		}

		fmt.Println("------------------------------")
		if len(trades) > 0 {
			fmt.Printf("exchange price => %2.f \n", trades[len(trades)-1].Price)
		}
		//dump.P(trades)
		fmt.Println("==============================")

		otherMarketSell := &client.PlaceOrderParams{
			UserID: 8,
			Bid:    false,
			Size:   1000,
		}

		orderResp, err := c.PlaceMarketOrder(otherMarketSell)
		if err != nil {
			log.Println(orderResp.OrderID)
		}

		marketSell := &client.PlaceOrderParams{
			UserID: 6,
			Bid:    false,
			Size:   1000,
		}

		orderResp, err = c.PlaceMarketOrder(marketSell)
		if err != nil {
			log.Println(orderResp.OrderID)
		}

		//-----------------------------------------
		otherMarketBuyOrder := &client.PlaceOrderParams{
			UserID: 8,
			Bid:    true,
			Size:   1000,
		}

		orderResp, err = c.PlaceMarketOrder(otherMarketBuyOrder)
		if err != nil {
			log.Println(orderResp.OrderID)
		}

		marketBuyOrder := &client.PlaceOrderParams{
			UserID: 6,
			Bid:    true,
			Size:   1000,
		}

		orderResp, err = c.PlaceMarketOrder(marketBuyOrder)
		if err != nil {
			log.Println(orderResp.OrderID)
		}

		<-ticker.C
	}
}

func makeMarketSimpel(c *client.Client) {
	ticker := time.NewTicker(tick)

	for {
		orders, err := c.GetOrders(userID)
		if err != nil {
			log.Println(err)
		}

		bestAsk, err := c.GetBestAsk()
		if err != nil {
			log.Println(err)
		}
		bestBid, err := c.GetBestBid()
		if err != nil {
			log.Println(err)
		}

		spread := math.Abs(bestBid - bestAsk)
		fmt.Println("exchange spread", spread)

		//place the bid
		fmt.Println("bid lens ", len(orders.Bids))
		if len(orders.Bids) < maxOrders {
			bidLimit := &client.PlaceOrderParams{
				UserID: userID,
				Bid:    true,
				Price:  bestBid + 100,
				Size:   1000,
			}

			bidOrderResp, err := c.PlaceLimitOrder(bidLimit)
			if err != nil {
				log.Println(bidOrderResp.OrderID)
			}
		}

		// place the ask
		fmt.Println("ask lens ", len(orders.Asks))
		if len(orders.Asks) < maxOrders {
			askLimit := &client.PlaceOrderParams{
				UserID: userID,
				Bid:    false,
				Price:  bestAsk - 100,
				Size:   1000,
			}

			askOrderResp, err := c.PlaceLimitOrder(askLimit)
			if err != nil {
				log.Println(askOrderResp.OrderID)
			}
		}

		fmt.Println("best ask price", bestAsk)
		fmt.Println("best bid price", bestBid)

		<-ticker.C
	}
}

func seedMarket(c *client.Client) error {
	ask := &client.PlaceOrderParams{
		UserID: 8,
		Bid:    false,
		Price:  10_000,
		Size:   1_000_000,
	}

	bid := &client.PlaceOrderParams{
		UserID: 8,
		Bid:    true,
		Price:  9_000,
		Size:   1_000_000,
	}

	_, err := c.PlaceLimitOrder(ask)
	if err != nil {
		return err
	}

	_, err = c.PlaceLimitOrder(bid)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	go server.StartServer()

	time.Sleep(1 * time.Second)

	c := client.NewClient()

	if err := seedMarket(c); err != nil {
		panic(err)
	}

	go makeMarketSimpel(c)

	time.Sleep(1 * time.Second)

	marketOrderPlacer(c)

	select {}
}
