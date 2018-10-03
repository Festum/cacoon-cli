package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/caarlos0/env"
	"github.com/jmoiron/jsonq"
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

type Cacoon struct {
	ApiKey string `env:"CACOON_API_KEY,required"`
	ApiEP  string `env:"CACOON_ENDPOINT" envDefault:"https://cacoo.com/api/v1"`
}

type diagram struct {
	URL            string `json:"url"`
	ImageURL       string `json:"imageUrl"`
	ImageURLForAPI string `json:"imageUrlForApi"`
	DiagramID      string `json:"diagramId"`
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	Security       string `json:"security"`
	Type           string `json:"type"`
	Owner          struct {
		Name     string `json:"name"`
		Nickname string `json:"nickname"`
		Type     string `json:"type"`
		ImageURL string `json:"imageUrl"`
	} `json:"owner"`
	OwnerName        string        `json:"ownerName"`
	OwnerNickname    string        `json:"ownerNickname"`
	Editing          interface{}   `json:"editing,omitempty"`
	Own              bool          `json:"own"`
	Shared           bool          `json:"shared"`
	FolderID         interface{}   `json:"folderId,omitempty"`
	FolderName       interface{}   `json:"folderName,omitempty"`
	ProjectID        interface{}   `json:"projectId,omitempty"`
	ProjectName      interface{}   `json:"projectName,omitempty"`
	OrganizationKey  interface{}   `json:"organizationKey,omitempty"`
	OrganizationName interface{}   `json:"organizationName,omitempty"`
	Created          string        `json:"created"`
	Updated          string        `json:"updated"`
	Sheets           []interface{} `json:"sheets,omitempty"`
	Comments         []interface{} `json:"comments,omitempty"`
}

type diagramResp struct {
	Result []diagram `josn:"result"`
	Count  int       `josn:"count"`
}

func main() {
	app := cli.NewApp()
	app.Name = "Cacoon"
	app.Usage = "A wrapper CLI for Cacoon"

	cc := Cacoon{}
	cc.init()

	app.Commands = []cli.Command{
		{
			Name:    "diagram",
			Aliases: []string{"d"},
			Usage:   "control diagram",
			Subcommands: []cli.Command{
				{
					Name:    "add",
					Aliases: []string{"a"},
					Usage:   "add a new diagram",
					Action: func(c *cli.Context) error {
						resp, err := http.Get(cc.ApiEP + "/diagrams/create.json?apiKey=" + cc.ApiKey)
						if err != nil {
							fmt.Println("API error ", err.Error())
						}
						fmt.Println("new diagram: ", resp)
						return nil
					},
				},
				{
					Name:    "list",
					Aliases: []string{"l"},
					Usage:   "List diagrams",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:   "ids, i",
							Hidden: false,
							Usage:  "List diagram ids",
						},
					},
					Action: func(c *cli.Context) error {
						resp, err := cc.api("diagrams", "", "")
						if err != nil {
							cli.NewExitError(err, 86)
						}
						body, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							cli.NewExitError(err, 86)
						}
						d := diagramResp{}
						if err := json.Unmarshal(body, &d); err != nil {
							cli.NewExitError(err, 86)
						}

						if c.Bool("ids") {
							lst := []string{}
							for _, r := range d.Result {
								lst = append(lst, r.DiagramID)
							}
							fmt.Printf("%+v\n", lst)
							return nil
						}
						if j, err := json.Marshal(d); err == nil {
							fmt.Printf("%+v\n", string(j))
						}
						return nil
					},
				},
				{
					Name:    "get",
					Aliases: []string{"g"},
					Usage:   "get diagrams",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "filter, f",
							Usage: "Filter the data",
						},
					},
					Action: func(c *cli.Context) error {
						did := c.Args().First()
						resp, err := cc.api("diagrams", did+"/delete", "")
						if err != nil {
							cli.NewExitError(err, 86)
						}
						if resp.StatusCode != http.StatusOK {
							if resp.StatusCode == http.StatusNotFound {
								fmt.Printf("No such Diagram ID \n")
								return nil
							}
							fmt.Printf("Unexpected error code %s\n", resp.StatusCode)
							return nil
						}
						body, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							cli.NewExitError(err, 86)
						}

						if len(c.String("filter")) > 0 {
							data := map[string]interface{}{}
							dec := json.NewDecoder(strings.NewReader(string(body)))
							dec.Decode(&data)
							jq := jsonq.NewQuery(data)
							v, err := jq.String(c.String("filter"))
							if err != nil {
								fmt.Printf("No such name %s in root payload\n", c.String("filter"))
								return nil
							}
							fmt.Printf("%s\n", v)
							return nil
						}
						// Check format with struct
						d := diagram{}
						if err := json.Unmarshal(body, &d); err != nil {
							cli.NewExitError(err, 86)
						}

						if j, err := json.Marshal(d); err == nil {
							fmt.Printf("%+v\n", string(j))
						}
						return nil
					},
				},
				{
					Name:    "remove",
					Aliases: []string{"r", "d"},
					Usage:   "remove an existing diagram",
					Action: func(c *cli.Context) error {
						did := c.Args().First()
						resp, err := cc.api("diagrams", did+"/delete", "")
						if err != nil {
							cli.NewExitError(err, 86)
						}
						if resp.StatusCode != http.StatusOK {
							if resp.StatusCode == http.StatusNotFound {
								fmt.Printf("No such Diagram ID \n")
								return nil
							}
							fmt.Printf("Unexpected error code %s\n", resp.StatusCode)
							return nil
						}
						fmt.Printf("%s has successfully deleted\n", did)
						return nil
					},
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func (cc *Cacoon) init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file: ", err)
	}

	if err := env.Parse(cc); err != nil {
		log.Fatalf("%+v\n", err)
	}
}

func (cc *Cacoon) api(target, action, body string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s.json?apiKey=%s", cc.ApiEP, target, cc.ApiKey)
	if action != "" {
		url = fmt.Sprintf("%s/%s/%s.json?apiKey=%s", cc.ApiEP, target, action, cc.ApiKey)
	}
	if body == "" {
		return http.Get(url)
	} else {
		var raw map[string]interface{}
		json.Unmarshal([]byte(body), &raw)
		jsonBody, _ := json.Marshal(raw)
		return http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	}
	return &http.Response{}, errors.New("Unknown Error")
}
