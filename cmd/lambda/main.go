package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	drive "github.com/iochen/msgraph-drive"
)

type Event struct {
	Body              string              `json:"body"`
	Path              string              `json:"path"`
	PathParameters    map[string]string   `json:"pathParameters"`
	HTTPMethod        string              `json:"httpMethod"`
	IsBase64Encoded   bool                `json:"isBase64Encoded"`
	Headers           map[string]string   `json:"headers"`
	MultiValueHeaders map[string][]string `json:"multiValueHeaders"`
}

type Response struct {
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	StatusCode      int               `json:"statusCode"`
	Body            string            `json:"body"`
	Headers         map[string]string `json:"headers"`
}

type Config struct {
	TenantID      string
	ApplicationID string
	ClientSecret  string
	DriveID       string
	View          string
	Listen        string
}

type DataItem struct {
	Name         string
	ReadableSize string
	Date         time.Time
	ReadableDate string
	IsFolder     bool
}

type Data struct {
	Parent  string
	Current string
	Items   []DataItem
}

var drv *drive.Drive
var tpl *template.Template

func init() {
	conf := &Config{
		TenantID:      os.Getenv("TENANT_ID"),
		ApplicationID: os.Getenv("APP_ID"),
		ClientSecret:  os.Getenv("CLI_SECRET"),
		View:          os.Getenv("DRV_VIEW"),
		DriveID:       os.Getenv("DRIVE_ID"),
	}
	cli, err := drive.NewGraphClient(conf.TenantID, conf.ApplicationID, conf.ClientSecret)
	if err != nil {
		log.Fatalln(err)
	}
	drv = cli.GetDrive(conf.DriveID)
	if len(conf.View) == 0 {
		conf.View = "https://gist.githubusercontent.com/iochen/87bcef49b24ef84957e16a5fdd6c0bd8/raw/drive.html"
	}
	resp, err := http.Get(conf.View)
	if err != nil {
		log.Fatalln(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	tpl, err = template.New("drive").Parse(string(body))
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	lambda.Start(Serve)
}

func Serve(event Event) (Response, error) {
	path := event.PathParameters["proxy"]
	items, err := drv.ListChildren(path)
	if err != nil {
		switch err.(type) {
		case *drive.ReqError:
			if err.(*drive.ReqError).Err.Code == "itemNotFound" {
				return Response{
					IsBase64Encoded: false,
					StatusCode:      http.StatusNotFound,
					Body:            "Item Not Found.",
				}, nil
			}
		}
		return Response{
			IsBase64Encoded: false,
			StatusCode:      http.StatusInternalServerError,
			Body:            "Internal Server Error.",
		}, err
	}

	if len(items) == 0 {
		item, err := drv.Item(path)
		if err != nil {
			return Response{
				IsBase64Encoded: false,
				StatusCode:      http.StatusInternalServerError,
				Body:            "Internal Server Error.",
			}, err
		}
		return Response{
			IsBase64Encoded: false,
			StatusCode:      http.StatusTemporaryRedirect,
			Headers: map[string]string{
				"Location": item.DownloadURL,
			},
		}, nil
	}

	parent := path
	if !strings.HasPrefix(parent, "/") {
		parent = "/" + parent
	}
	if strings.HasSuffix(parent, "/") {
		parent = parent[:len(parent)-1]
	}
	parent = filepath.Dir(parent)
	data := Data{
		Parent:  parent,
		Current: path,
		Items:   []DataItem{},
	}
	for i := range items {
		data.Items = append(data.Items, DataItem{
			Name:         items[i].Name,
			ReadableSize: size2readable(items[i].Size),
			Date:         items[i].LastMod,
			ReadableDate: date2readable(items[i].LastMod),
			IsFolder:     items[i].IsFolder(),
		})
	}
	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, data)
	if err != nil {
		return Response{
			IsBase64Encoded: false,
			StatusCode:      http.StatusInternalServerError,
			Body:            "Internal Server Error.",
		}, err
	}
	return Response{
		IsBase64Encoded: false,
		StatusCode:      http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Body: buf.String(),
	}, nil
}

func date2readable(date time.Time) string {
	sub := time.Now().Sub(date)
	hours := sub.Hours()
	minutes := sub.Minutes()
	switch {
	case hours < 1:
		switch {
		case minutes < 1:
			return "recently"
		default:
			return fmt.Sprintf("%.0f minute(s) ago", minutes)
		}
	case hours < 24:
		return fmt.Sprintf("%.0f hour(s) ago", hours)
	default:
		return fmt.Sprintf("%.0f day(s) ago", hours/24)
	}
}

func size2readable(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(size)/float64(div), "KMGTPE"[exp])
}
