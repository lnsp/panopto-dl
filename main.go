package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	id       = flag.String("id", "", "Stream ID to download")
	output   = flag.String("o", "index.mp4", "Destination file")
	folder   = flag.Bool("f", false, "Download entire folder")
	aspx     = flag.String("a", "", "ASPX auth token")
	delivery = flag.String("delivery", "https://tum.cloud.panopto.eu/Panopto/Pages/Viewer/DeliveryInfo.aspx", "DeliveryInfo URL")
	session  = flag.String("sessions", "https://tum.cloud.panopto.eu/Panopto/Services/Data.svc/GetSessions", "GetSessions URL")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	if *folder {
		return fetchFolder(*id)
	}
	url, err := extractStreamURL(*id)
	if err != nil {
		return fmt.Errorf("extract stream URL: %w", err)
	}
	if err := fetchStream(url, *output); err != nil {
		return fmt.Errorf("fetch stream: %w", err)
	}
	return nil
}

func fetchFolder(id string) error {
	folderReqBody := map[string]interface{}{
		"queryParameters": map[string]interface{}{
			"folderID":      id,
			"getFolderData": "true",
			"maxResults":    100,
		},
	}
	folderReqBodyData, _ := json.Marshal(folderReqBody)
	folderReq, err := http.NewRequest(http.MethodPost, *session, bytes.NewBuffer(folderReqBodyData))
	if err != nil {
		return fmt.Errorf("create list session req: %w", err)
	}
	folderReq.Header.Add("Content-Type", "application/json")
	folderReq.AddCookie(&http.Cookie{
		Name:  ".ASPXAUTH",
		Value: *aspx,
	})
	folderResp, err := http.DefaultClient.Do(folderReq)
	if err != nil {
		return fmt.Errorf("list session: %w", err)
	}
	defer folderResp.Body.Close()
	var folderRespJSON struct {
		Data struct {
			Results []struct {
				Name string `json:"SessionName"`
				ID   string `json:"DeliveryID"`
			} `json:"Results"`
		} `json:"d"`
	}
	decoder := json.NewDecoder(folderResp.Body)
	if err := decoder.Decode(&folderRespJSON); err != nil {
		return fmt.Errorf("decode resp: %w", err)
	}
	// Create destination folder
	if err := os.MkdirAll(*output, 0755); err != nil {
		return fmt.Errorf("create output folder: %w", err)
	}
	// Download sessions
	for _, session := range folderRespJSON.Data.Results {
		fmt.Fprintln(os.Stderr, "Downloading session", session.Name, session.ID)
		url, err := extractStreamURL(session.ID)
		if err != nil {
			return fmt.Errorf("extract stream: %w", err)
		}
		if err := fetchStream(url, filepath.Join(*output, session.Name+".mp4")); err != nil {
			return fmt.Errorf("fetch stream: %w", err)
		}
	}
	return nil
}

func fetchStream(url, destination string) error {
	// Write cookies to temporary file
	tmpCookieJar, err := ioutil.TempFile("", "cookies")
	if err != nil {
		return fmt.Errorf("create tmp cookie: ")
	}
	defer tmpCookieJar.Close()
	fmt.Fprintln(tmpCookieJar, "# Netscape HTTP Cookie File")
	fmt.Fprintln(tmpCookieJar, strings.Join([]string{
		"tum.cloud.panopto.eu",
		"FALSE",
		"/",
		"TRUE",
		"0",
		".ASPXAUTH",
		*aspx,
	}, "\t"))
	tmpCookieJar.Close()
	// Start youtube-dl
	cmd := exec.Command("youtube-dl", "--cookies", tmpCookieJar.Name(), "-f", "best", url, "-o", destination)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, string(output))
		return fmt.Errorf("download video: %w", err)
	}
	return nil
}

func extractStreamURL(id string) (string, error) {
	// Request DeliveryInfo.aspx
	deliveryBody := &url.Values{}
	deliveryBody.Add("deliveryId", id)
	deliveryBody.Add("responseType", "json")
	deliveryReq, err := http.NewRequest(http.MethodPost, *delivery, strings.NewReader(deliveryBody.Encode()))
	if err != nil {
		return "", fmt.Errorf("create delivery req: %w", err)
	}
	deliveryReq.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	deliveryReq.AddCookie(&http.Cookie{
		Name:  ".ASPXAUTH",
		Value: *aspx,
	})
	deliveryResp, err := http.DefaultClient.Do(deliveryReq)
	if err != nil {
		return "", fmt.Errorf("do delivery req: %w", err)
	}
	defer deliveryResp.Body.Close()
	// Parse delivery body
	var deliveryRespJSON struct {
		Delivery struct {
			Streams []struct {
				URL string `json:"StreamUrl"`
			} `json:"Streams"`
		} `json:"Delivery"`
	}
	decoder := json.NewDecoder(deliveryResp.Body)
	if err := decoder.Decode(&deliveryRespJSON); err != nil {
		return "", fmt.Errorf("decode delivery resp: %w", err)
	}
	// Pick first stream URL
	if len(deliveryRespJSON.Delivery.Streams) < 1 {
		return "", fmt.Errorf("no available streams")
	}
	return deliveryRespJSON.Delivery.Streams[0].URL, nil
}
