package upload

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/codersrank-org/multi_repo_repo_extractor/upload/entity"
	"github.com/pkg/browser"
)

// CodersrankService uploads and merge results with codersrank
type CodersrankService interface {
	UploadRepo(repoID string) (string, error)
	UploadResults(results map[string]string) string
	ProcessResults(resultToken string)
}

type codersrankService struct {
	UploadRepoURL   string
	UploadResultURL string
	ProcessURL      string
}

// NewCodersrankService constructor
func NewCodersrankService() CodersrankService {
	return &codersrankService{
		UploadRepoURL:   "https://grpcgateway.codersrank.io/candidate/privaterepo/Upload",
		UploadResultURL: "https://grpcgateway.codersrank.io/multi/repo/results",
		ProcessURL:      "https://profile.codersrank.io/repo?multiToken=",
	}
}

func (c *codersrankService) UploadRepo(repoID string) (string, error) {

	// Read file
	filename := fmt.Sprintf("%s/%s.zip", getSaveResultPath(), repoID)
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Add file as multipart/form-data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	if err != nil {
		return "", err
	}
	io.Copy(part, file)
	writer.Close()

	// Create and make the request
	request, err := http.NewRequest("POST", c.UploadRepoURL, body)
	if err != nil {
		return "", err
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", errors.New("Server returned non 200 response")
	}
	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	// Get response and return resulting token
	var result entity.UploadResult
	err = json.Unmarshal(content, &result)
	if err != nil {
		return "", err
	}

	return result.Token, nil
}

func (c *codersrankService) UploadResults(results map[string]string) string {

	multiUpload := entity.MultiUpload{}
	multiUpload.Results = make([]entity.UploadResultWithRepoName, len(results))

	i := 0
	for reponame, token := range results {
		multiUpload.Results[i] = entity.UploadResultWithRepoName{
			Token:    token,
			Reponame: reponame,
		}
		i++
	}

	b, err := json.Marshal(multiUpload)
	if err != nil {
		log.Fatal(err)
	}
	req, err := http.NewRequest("POST", c.UploadResultURL, bytes.NewBuffer(b))
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var result entity.UploadResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatal(err)
	}

	return result.Token

}

func (c *codersrankService) ProcessResults(resultToken string) {
	browserURL := c.ProcessURL + resultToken
	browser.OpenURL(browserURL)
}

func getSaveResultPath() string {
	appPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	resultPath := appPath + "/results"
	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		os.Mkdir(resultPath, 0700)
	}
	return resultPath
}