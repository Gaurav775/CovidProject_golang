package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Regional struct {
	Loc                   string `json:"loc" bson:"loc"`
	ConfirmedCasesIndian  int    `json:"confirmedCasesIndian" bson:"confirmedCasesIndian"`
	ConfirmedCasesForeign int    `json:"confirmedCasesForeign" bson:"confirmedCasesForeign"`
	Discharged            int    `json:"discharged" bson:"discharged"`
	Deaths                int    `json:"deaths" bson:"deaths"`
	TotalConfirmed        int    `json:"totalConfirmed" bson:"totalConfirmed"`
}

type Summary struct {
	Total                            int `json:"total" bson:"total"`
	ConfirmedCasesIndian             int `json:"confirmedCasesIndian" bson:"confirmedCasesIndian"`
	ConfirmedCasesForeign            int `json:"confirmedCasesForeign" bson:"confirmedCasesForeign"`
	Discharged                       int `json:"discharged" bson:"discharged"`
	Deaths                           int `json:"deaths" bson:"deaths"`
	DonfirmedButLocationUnidentified int `json:"confirmedButLocationUnidentified" bson:"confirmedButLocationUnidentified"`
}
type Data struct {
	Summary  Summary    `json:"summary" bson:"summary"`
	Regional []Regional `json:"regional" bson:"regional"`
}

type Response struct {
	Success          string `json:"success" bson:"success"`
	Data             Data   `json:"data" bson:"data"`
	LastRefreshed    string `json:"lastRefreshed" bson:"lastRefreshed"`
	LastOriginUpdate string `json:"lastOriginUpdate" bson:"lastOriginUpdate"`
}

type AddressResponse struct {
	Address Address `json:"address" bson:"address"`
}

type Address struct {
	State string `json:"state" bson:"state"`
}

const locationiqKey string = "pk.117fca37fdab22ec59d7f02c01ba89c1"

func main() {
	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "If you are able to see this line , setup successfully")
	})
	// fetching whole data of covid19 india
	response, err := http.Get("https://api.rootnet.in/covid19-in/stats/latest")

	if err != nil {
		fmt.Print(err.Error())
		os.Exit(0)
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var responseObject Response
	json.Unmarshal(responseData, &responseObject)

	// summary of corona cases
	e.GET("/summary", func(c echo.Context) error {
		return c.JSON(http.StatusOK, responseObject.Data.Summary)
	})

	// summary of each state of india
	e.GET("/regional", func(c echo.Context) error {
		return c.JSON(http.StatusOK, responseObject.Data.Regional)
	})

	// summary of particular state of india
	e.GET("/regional/:loc", func(c echo.Context) error {

		for idx, regional := range responseObject.Data.Regional {
			if regional.Loc == c.Param("loc") {
				fmt.Print(idx)
				return c.JSON(http.StatusOK, regional)
			}
		}

		return c.JSON(http.StatusNotFound, echo.ErrNotFound)
	})

	// summary of particular state of india using its lon and lat
	// lon and lat passed as query params
	e.GET("/regional/", func(c echo.Context) error {
		lon := c.QueryParam("lon")
		lat := c.QueryParam("lat")

		// url of locationiq.com to fetch location info
		url := "https://us1.locationiq.com/v1/reverse.php?key=" + locationiqKey + "&lat=" + lat + "&lon=" + lon + "&format=json"
		response, err := http.Get(url)

		if err != nil {
			fmt.Print(err.Error())
			os.Exit(0)
		}

		responseData, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}

		var AddressResponseObj AddressResponse
		json.Unmarshal(responseData, &AddressResponseObj)

		correspondingState := AddressResponseObj.Address.State

		for idx, regional := range responseObject.Data.Regional {
			if regional.Loc == correspondingState {
				fmt.Print(idx)
				return c.JSON(http.StatusOK, regional)

			}
		}
		return c.JSON(http.StatusNotFound, echo.ErrNotFound)
	})

	// connection with mongo db
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(databases)

	db := client.Database("CoronaApp")

	collection := db.Collection("CoronaData")

	res, err := collection.InsertOne(context.Background(), responseObject)

	fmt.Print(collection.Name())

	e.GET("/collectionname", func(c echo.Context) error {
		return c.String(http.StatusOK, collection.Name())
	})

	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(res.InsertedID)

	defer client.Disconnect(ctx)

	// starting server
	e.Logger.Fatal(e.Start(":8080"))

}
