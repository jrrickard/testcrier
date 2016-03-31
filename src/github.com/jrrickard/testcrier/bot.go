package main

import (
	"encoding/xml"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/nlopes/slack"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var summarySender chan *TestSummary

var slackChannels map[string]string = nil

type Response struct {
	Text    string `json:"text,omitempty"`
	Channel string
	Params  *slack.PostMessageParameters
}

type Testsuite struct {
	Name      string      `xml:"name,attr"`
	Tests     int         `xml:"tests,attr"`
	Failures  int         `xml:"failures,attr"`
	Errors    int         `xml:"errors,attr"`
	Timestamp string      `xml:"timestamp,attr"`
	Time      float64     `xml:"time,attr"`
	Hostname  string      `xml:"hostname,attr"`
	Testcases []*TestCase `xml:"testcase"`
}

type TestCase struct {
	Name      string    `xml:"name,attr"`
	Time      float64   `xml:"time,attr"`
	Classname string    `xml:"classname,attr"`
	Failure   *Failure  `xml:"failure"`
	Skipped   *struct{} `xml:"skipped"`
}

type Failure struct {
	Type    string `xml:"type,attr"`
	message string `xml:"message,attr"`
}

type TestSummary struct {
	Destination string
	Name        string
	Tests       int
	Failures    int
	Skipped     int
}

func handleTests(testSuite *Testsuite, destination string) {
	fmt.Println(testSuite.Name)
	skipped := 0

	for _, test := range testSuite.Testcases {
		testCase := *test
		if testCase.Skipped != nil {
			skipped++
		}
	}


	testSummary := TestSummary{Destination: destination, Name: testSuite.Name, Tests: testSuite.Tests, Failures: testSuite.Failures, Skipped: skipped}
	go func(testSummary TestSummary) {
		summarySender <- &testSummary
	}(testSummary)

}

func HandleTestSuite(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("uploadfile")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	bytes, _ := ioutil.ReadAll(file)

	var testSuite Testsuite
	err = xml.Unmarshal(bytes, &testSuite)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	channelName := r.URL.Query().Get("channel")
	if channelName == "" {
		channelName = os.Getenv("CHANNEL")
	}
	destination, foundIt := slackChannels[channelName]
	if foundIt {
		handleTests(&testSuite, destination)
	} else {
		fmt.Printf("Received a test suite but don't know where to send it")
	}
}

func generateMessage(testSummary TestSummary) (*Response, error) {
	response := Response{}
	params := slack.PostMessageParameters{}
	params.AsUser = true
	passed := testSummary.Tests - testSummary.Failures - testSummary.Skipped
	status := "Passed"
	if testSummary.Failures > 0 {
		status = "Failed"
	}
	response.Text = fmt.Sprintf("*Test Results For %s*", testSummary.Name)
	attachment := slack.Attachment{
		Color:      generateColorTestStatus(testSummary.Failures),
		MarkdownIn: []string{"text", "pretext", "fields"},
		Fields: []slack.AttachmentField{
			slack.AttachmentField{
				Title: "Result",
				Value: status,
			},
			slack.AttachmentField{
				Title: "Total",
				Value: fmt.Sprintf("%v", testSummary.Tests),
				Short: true,
			},
			slack.AttachmentField{
				Title: "Passed",
				Value: fmt.Sprintf("%v", passed),
				Short: true,
			},
			slack.AttachmentField{
				Title: "Failed",
				Value: fmt.Sprintf("%v", testSummary.Failures),
				Short: true,
			},
			slack.AttachmentField{
				Title: "Skipped",
				Value: fmt.Sprintf("%v", testSummary.Skipped),
				Short: true,
			},
		},
	}
	params.Attachments = []slack.Attachment{attachment}
	fmt.Printf("Sending slack message to : %s \n", testSummary.Destination)
	response.Channel = testSummary.Destination
	response.Params = &params
	return &response, nil

}

func generateColorTestStatus(failures int) string {
	if failures > 0 {
		return "#FF0000"
	} else {
		return "#008000"
	}
}

func main() {

	slackChannels = make(map[string]string)
	api := slack.New(os.Getenv("SLACK_TOKEN"))
	_, err := api.AuthTest()
	if err != nil {
		fmt.Printf("Unable to login to the API\n")
		fmt.Errorf("%s\n", err)
		os.Exit(1)
	}
	summarySender = make(chan *TestSummary)

	api.SetUserAsActive()
	go func(api *slack.Client, summarySender chan *TestSummary) {
		for {
			select {
			case testSummary := <-summarySender:
				response, errors := generateMessage(*testSummary)
				if errors != nil {
					fmt.Printf("Unable to send the slack message %v\n", errors)
				}
				destination := response.Channel
				if strings.HasPrefix(destination, "U") {
					_, _, imID, _ := api.OpenIMChannel(destination)
					destination = imID
				}
				api.PostMessage(destination, response.Text, *response.Params)
			}
		}

	}(api, summarySender)

	channels, err := api.GetChannels(true)
	if err != nil {
		fmt.Errorf("%s\n", err)
		os.Exit(1)
	}
	for _, element := range channels {
		slackChannels[element.Name] = element.ID
	}

	groups, err := api.GetGroups(true)
	if err != nil {
		fmt.Errorf("%s\n", err)
		os.Exit(1)
	}
	for _, element := range groups {
		slackChannels[element.Name] = element.ID
	}

	router := httprouter.New()
	router.POST("/test/:name", HandleTestSuite)
	log.Fatal(http.ListenAndServe(":8080", router))
}
