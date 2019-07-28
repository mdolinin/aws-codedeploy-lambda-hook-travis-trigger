package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type BuildRequest struct {
	Request Request `json:"request"`
}

type Request struct {
	Id      int     `json:"id,omitempty"`
	Message string  `json:"message,omitempty"`
	Branch  string  `json:"branch,omitempty"`
	Config  Config  `json:"config,omitempty"`
	Builds  []Build `json:"builds,omitempty"`
}

type Config struct {
	MergeMode []string `json:"merge_mode"`
	Script    string   `json:"script"`
	Deploy    Deploy   `json:"deploy"`
}

type Deploy struct {
	Script string `json:"script"`
}

type Build struct {
	Id    int    `json:"id,omitempty"`
	State string `json:"state,omitempty"`
}

type BuildResponse struct {
	Request Request `json:"request"`
}

func main() {
	startTime := time.Now()
	fmt.Println("Started Time :", startTime.String())
	travisCiToken := os.Getenv("TRAVIS_CI_TOKEN")
	if travisCiToken == "" {
		log.Fatalln("TRAVIS_CI_TOKEN is not set!")
	}
	githubRepoOwner := os.Getenv("GITHUB_REPO_OWNER")
	if githubRepoOwner == "" {
		log.Fatalln("GITHUB_REPO_OWNER is not set!")
	}
	githubRepoName := os.Getenv("GITHUB_REPO_NAME")
	if githubRepoName == "" {
		log.Fatalln("GITHUB_REPO_NAME is not set!")
	}
	branchName := os.Getenv("BRANCH_NAME")
	if branchName == "" {
		branchName = "master"
	}
	requestId := startTravisBuild(travisCiToken, githubRepoOwner, githubRepoName, branchName)
	testsPassed := getBuildResults(requestId, travisCiToken, githubRepoOwner, githubRepoName)
	fmt.Printf("E2E test execution on Travis is passed - %t\n", testsPassed)
	finishTime := time.Now()
	fmt.Println("Finish Time :", finishTime.String())
	fmt.Println("Duration Time :", finishTime.Sub(startTime).String())
}

func getBuildResults(requestId int, travisCiToken string, githubRepoOwner string, githubRepoName string) bool {
	testsPassed := false
	getRequestUrl := fmt.Sprintf("https://api.travis-ci.com/repo/%s%%2F%s/request/%d", githubRepoOwner, githubRepoName, requestId)
	fmt.Println(getRequestUrl)
	client := &http.Client{}
	request, err := http.NewRequest("GET", getRequestUrl, nil)
	if err != nil {
		log.Fatalln(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Travis-API-Version", "3")
	request.Header.Set("Authorization", fmt.Sprintf("token %s", travisCiToken))
	for i := 0; i <= 20; i++ {
		resp, err := client.Do(request)
		if err != nil {
			log.Fatalln(err)
		}
		defer func() {
			err := resp.Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		}()
		result := Request{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(result)
		if len(result.Builds) > 0 {
			build := result.Builds[0]
			buildId := build.Id
			state := build.State
			fmt.Printf("Travis build %d is %s\n", buildId, state)
			if state == "passed" {
				testsPassed = true
				break
			} else if state == "errored" {
				testsPassed = false
				break
			}
		}
		time.Sleep(60 * time.Second)
	}
	return testsPassed
}

func startTravisBuild(travisCiToken string, githubRepoOwner string, githubRepoName string, branchName string) int {
	buildRequest := BuildRequest{
		Request: Request{
			Message: "This debug of AWS B/G Deploy Lambda Hook",
			Branch:  branchName,
			Config: Config{
				MergeMode: []string{"replace"},
				Script:    "echo GO build",
				Deploy: Deploy{
					Script: "echo GO deploy",
				},
			},
		},
	}
	requestBody, err := json.Marshal(buildRequest)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(string(requestBody))
	client := &http.Client{}
	postRequestUrl := fmt.Sprintf("https://api.travis-ci.com/repo/%s%%2F%s/requests", githubRepoOwner, githubRepoName)
	fmt.Println(postRequestUrl)
	request, err := http.NewRequest("POST", postRequestUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatalln(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Travis-API-Version", "3")
	request.Header.Set("Authorization", fmt.Sprintf("token %s", travisCiToken))
	resp, err := client.Do(request)
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	result := BuildResponse{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(result)
	requestId := result.Request.Id
	fmt.Printf("Travis request id is %d\n", requestId)
	return requestId
}
