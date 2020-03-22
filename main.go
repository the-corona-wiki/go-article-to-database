package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/jinzhu/gorm"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Stores the db value globally to be called
var dB *gorm.DB
var mux sync.Mutex
var clientDB *mongo.Client

type qOptions struct {
	Q     string `url:"q"`
	Begin string `url:"begin_date"`
	End   string `url:"end_date"`
	Key   string `url:"api-key"`
}

type articles struct {
	Status    string   `json:"status"`
	Copyright string   `json:"copyright"`
	Response  response `json:"response"`
}

type multimedia struct {
	Rank    int         `json:"rank"`
	Subtype string      `json:"subtype"`
	Caption interface{} `json:"caption"`
	Credit  interface{} `json:"credit"`
	Type    string      `json:"type"`
	URL     string      `json:"url"`
	Height  int         `json:"height"`
	Width   int         `json:"width"`

	SubType  string `json:"subType"`
	CropName string `json:"crop_name"`
}
type headline struct {
	Main          string      `json:"main"`
	Kicker        interface{} `json:"kicker"`
	ContentKicker interface{} `json:"content_kicker"`
	PrintHeadline interface{} `json:"print_headline"`
	Name          interface{} `json:"name"`
	Seo           interface{} `json:"seo"`
	Sub           interface{} `json:"sub"`
}
type keywords struct {
	gorm.Model

	Name  string `json:"name"`
	Value string `json:"value"`
	Rank  int    `json:"rank"`
	Major string `json:"major"`
}
type person struct {
	Firstname    string      `json:"firstname"`
	Middlename   interface{} `json:"middlename"`
	Lastname     string      `json:"lastname"`
	Qualifier    interface{} `json:"qualifier"`
	Title        interface{} `json:"title"`
	Role         string      `json:"role"`
	Organization string      `json:"organization"`
	Rank         int         `json:"rank"`
}
type byline struct {
	Original     string      `json:"original"`
	Person       []person    `json:"person"`
	Organization interface{} `json:"organization"`
}
type docs struct {
	Abstract       string       `json:"abstract"`
	WebURL         string       `json:"web_url"`
	Snippet        string       `json:"snippet"`
	LeadParagraph  string       `json:"lead_paragraph"`
	Source         string       `json:"source"`
	Multimedia     []multimedia `json:"multimedia"`
	Headline       headline     `json:"headline"`
	Keywords       []keywords   `json:"keywords"`
	PubDate        string       `json:"pub_date"`
	DocumentType   string       `json:"document_type"`
	NewsDesk       string       `json:"news_desk"`
	SectionName    string       `json:"section_name"`
	Byline         byline       `json:"byline"`
	TypeOfMaterial string       `json:"type_of_material"`
	ID             string       `json:"_id"`
	WordCount      int          `json:"word_count"`
	URI            string       `json:"uri"`
	PrintSection   string       `json:"print_section,omitempty"`
	PrintPage      string       `json:"print_page,omitempty"`
	SubsectionName string       `json:"subsection_name,omitempty"`
}
type meta struct {
	Hits   int `json:"hits"`
	Offset int `json:"offset"`
	Time   int `json:"time"`
}
type response struct {
	Docs []docs `json:"docs"`
	Meta meta   `json:"meta"`
}

type articleJSON struct {
	gorm.Model

	Abstract  string     `json:"abstract"`
	URL       string     `json:"url"`
	Title     string     `json:"title"`
	Byline    string     `json:"byline"`
	Published string     `json:"published"`
	Keywords  []keywords `json:"keywords"`
}

func dateArray(beginY, beginM, beginD, endY, endM, endD int) []string {
	start := time.Date(beginY, time.Month(beginM), beginD, 0, 0, 0, 0, time.UTC)
	end := time.Date(endY, time.Month(endM), endD, 0, 0, 0, 0, time.UTC)

	var timeArray []string
	timeArray = append(timeArray, strings.ReplaceAll(start.Format("2006-01-02"), "-", ""))

	for start != end {
		start = start.AddDate(0, 0, 1)
		timeArray = append(timeArray, strings.ReplaceAll(start.Format("2006-01-02"), "-", ""))
	}
	return timeArray
}

func getAllArticles(dateArray []string) map[string]interface{} {
	articleMap := make(map[string]interface{})

	for _, date := range dateArray {
		articleMap[date] = apiReq(date)
	}

	return articleMap
}

func apiReq(date string) []articleJSON {
	var artArray []articleJSON
	opt := qOptions{Q: "coronavirus", Begin: date, End: date, Key: "oBMAtGUkfBI6JwVRXlW0M1Pk7dAGKdST"}

	v, err := query.Values(opt)
	check(err)

	resp, err := http.Get("https://api.nytimes.com/svc/search/v2/articlesearch.json?" + v.Encode())
	check(err)

	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	var articles articles

	err = json.Unmarshal(body, &articles)
	check(err)
	// if len(articles.Response.Docs) == 0 {
	// 	return artArray
	// }
	fmt.Println(len(articles.Response.Docs))

	for _, article := range articles.Response.Docs {
		art := articleJSON{URL: article.WebURL, Abstract: article.Abstract, Title: article.Headline.Main, Byline: article.Byline.Original, Published: article.PubDate, Keywords: article.Keywords}
		artArray = append(artArray, art)
		openDBandInsertArticle(clientDB, date, art)
	}

	return artArray
}

func jsonify(articleMap map[string]interface{}) []byte {
	json, err := json.MarshalIndent(articleMap, "", "	")
	if err != nil {
		panic(err)
	}
	return json
}

func writeJSONFile(json []byte) {
	err := ioutil.WriteFile("output.json", json, 0644)
	if err != nil {
		panic(err)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func openDBandInsertArticle(client *mongo.Client, date string, article articleJSON) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	collection := client.Database("articles").Collection(date)
	res, err := collection.InsertOne(ctx, article)
	check(err)
	fmt.Println(res)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(
		"mongodb+srv://omar:Corona1234@cluster0-3rny3.mongodb.net/test?retryWrites=true&w=majority",
	))
	check(err)

	clientDB = client
	// collection := client.Database("testing").Collection("numbers")

	// opts := options.FindOne()

	getAllArticles(dateArray(2019, 12, 01, 2020, 01, 12))

}
