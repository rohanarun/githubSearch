//Copyright 2017 Rohan Arun

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"io/ioutil"
	"strings"
	"reflect"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
	"net/http"
	"gopkg.in/olivere/elastic.v5"
)

// GitData contains the returned JSON response from the Github REST API for a repo tree of folders and files.
type GitData struct {
	Path 	string 	`json:"path"`
	Mode 	string 	`json:"mode"`
	Typ 	string 	`json:"type"`
	Sha 	string 	`json:"sha"`
	Size 	int64 	`json:"size"`
	Url 	string 	`json:"url"`
}

//GitAPIResponse contains the base JSON response from the Github REST API.
type GitAPIResponse struct {
	Sha 		string 		`json:"sha"`
	Url 		string 		`json:"url"`
	Tree		[]GitData 	`json:"tree"`
	Trunc 		bool 		`json:"truncated"`
}

//Repo is the ElasticSearch structure used to store files in the database.
type Repo struct {
	Name     string		`json:"name"`
	Message  string  	`json:"message"`
}

//Mapping is the ElasticSearch mapping paramerters.
const mapping = `
{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
		"repo":{
			"properties":{
				"name":{
					"type":"keyword"
				},
				"message":{
					"type":"text",
					"store": true,
					"fielddata": true,
					"analyzer": "whitespace"
				}
			}
		}
	}
}`

var dbName = "default"
var indexDB = "default"
func main() {
	r := gin.Default()
	m := melody.New()

	ctx := context.Background()
	client, err := elastic.NewClient()
	if err != nil {
		fmt.Println("Error with elastic.NewClient: ", err)
	}

	//Serves the index.html page over port 5000 or through nginx reverse proxy
	r.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})


	r.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})

	//Handles message sent from frontend through websockets
	m.HandleMessage(func(s *melody.Session, msg []byte) {

		input :=  string(msg[:])
		isRepo := strings.Contains(input, "repo:")
		//If the message is a repo name, perform indexing using the Github REST API. Otherwise, perform a search
		//TODO: make indexing and searching concurrent.
		if isRepo == false{
                        searchTerm := input[7:len(input)]

			// Flush to make sure the documents got written.
			_, err = client.Flush().Index(dbName).Do(ctx)
			if err != nil {
                                        fmt.Println("Error with Flush(): ", err)
			}

			// Search with a term query
			termQuery := elastic.NewMatchPhrasePrefixQuery("message", searchTerm)
			searchResult, err := client.Search().
			Index(dbName).   // search in index "twitter"
			Query(termQuery).   // specify the query
			Sort("name", true). // sort by "user" field, ascending
			From(0).Size(100).   // take documents 0-9
			Pretty(true).       // pretty print request and response JSON
			Do(ctx)             // execute
			if err != nil {
                                        fmt.Println("Error with Search(): ", err)
			}

			fmt.Printf("Query took %d milliseconds\n", searchResult.TookInMillis)
			var ttyp Repo
			for _, item := range searchResult.Each(reflect.TypeOf(ttyp)) {
				if t, ok := item.(Repo); ok {
					searchTime := strconv.Itoa(int(searchResult.TookInMillis))
					//Send back the search results
					mySlice := []byte("Search Time: " + searchTime + " milliseconds<br> File Name: " + t.Name + " Elastic Text: \n" + t.Message)
					m.Broadcast(mySlice)
				}
			}
		}
		if isRepo == true {
			repoName := input[5:len(input)]
			//Use the Github API to get all the files in the repo 
			resp, err := http.Get("https://api.github.com/repos/"+ repoName+"/git/trees/master?recursive=100")
			if err != nil {
				fmt.Println("Error with http.get(): ", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
        			bodyBytes, err := ioutil.ReadAll(resp.Body)
                                if err != nil {
                                        fmt.Println("Error with ioutil.ReadAll: ", err)
                                }
				var GithubResponse = new(GitAPIResponse)
    				err = json.Unmarshal([]byte(bodyBytes), &GithubResponse)
   	 			if(err != nil){
        				fmt.Println("Error with json.unmarshal: ", err)
    				}
				TreeData := GithubResponse.Tree
                        	dbName = strings.Replace( repoName, "/", ".", -1)

				exists, err := client.IndexExists(dbName).Do(ctx)
                        	if err != nil {
                                        fmt.Println("Error with IndexExists: ", err)
                        	}
                        	if !exists {
                        		fmt.Println("Creating new db for :", repoName)
                                	_, err := client.CreateIndex(dbName).BodyString(mapping).Do(ctx)
                               		if err != nil {
 	                                       fmt.Println("Error with Creating new ElasticSearch Index: ", err)
                                	}
                    		}


				//Traverse all the files in the github tree, and perform indexing
				for index, element := range TreeData {
					//Get the raw file contents of github files, and make sure not to load images
					if element.Size < 50000 && !strings.Contains(element.Path, ".gif") && !strings.Contains(element.Path, ".png")  && !strings.Contains(element.Path, ".ico")  && !strings.Contains(element.Path, ".jpg")  && !strings.Contains(element.Path, ".jpeg") {
                				resp, err := http.Get("https://raw.githubusercontent.com/"+repoName+"/master/"+element.Path)
                				if err != nil {
	               		                         fmt.Println("Error with http.get(): ", err)
               		 			}
        	        			defer resp.Body.Close()
						//If the file exists, index it using Elastic Search
						//TODO: make indexing parallel
                				if resp.StatusCode == 200 { // OK
                       		 			bodyBytes, err := ioutil.ReadAll(resp.Body)
               		 		        	var bodyString = string(bodyBytes)
  							if err != nil {
 			                                       fmt.Println("Error with ioutil.ReadAll: ", err)
       		                         		}
							repo1 := Repo{Name: element.Path, Message: bodyString}
							put1, err := client.Index().
							Index(dbName).
							Type("repo").
							Id(string(index)).
							BodyJson(repo1).
							Do(ctx)
							if err != nil {
	 		                                       fmt.Println("Error with client.Index(): ", err)
							}
							//After files are indexed, send the file name and contents back to the frontend to build the file explorer
							mySlice := []byte("newFilePath:" + element.Path + " newFileBody:" + bodyString);
							m.Broadcast(mySlice);
							fmt.Printf("Indexed file %s to index %s, type %s\n", put1.Id, put1.Index, put1.Type)

						}
					}

				}
				//Debug info sent back to frontend
    				m.Broadcast(bodyBytes)
			}
		}
	})

	r.Run(":5000")
}
