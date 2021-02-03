package drive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Client represents a msgraph API connection instance.
//
// An instance can also be json-unmarshalled an will immediately be initialized, hence a Token will be
// grabbed. If grabbing a token fails the JSON-Unmarshal returns an error.
type Client struct {
	sync.Mutex // lock it when performing an API-call to synchronize it

	TenantID      string // See https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal#get-tenant-id
	ApplicationID string // See https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal#get-application-id-and-authentication-key
	ClientSecret  string // See https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal#get-application-id-and-authentication-key

	token Token // the current token to be used
}

func (cli *Client) String() string {
	var firstPart, lastPart string
	if len(cli.ClientSecret) > 4 { // if ClientSecret is not initialized prevent a panic slice out of bounds
		firstPart = cli.ClientSecret[0:3]
		lastPart = cli.ClientSecret[len(cli.ClientSecret)-3:]
	}
	return fmt.Sprintf("Client(TenantID: %v, ApplicationID: %v, ClientSecret: %v...%v, Token validity: [%v - %v])",
		cli.TenantID, cli.ApplicationID, firstPart, lastPart, cli.token.NotBefore, cli.token.ExpiresOn)
}

// NewGraphClient creates a new Client instance with the given parameters and grab's a token.
//
// Returns an error if the token can not be initialized. This method does not have to be used to create a new Client
func NewGraphClient(tenantID, applicationID, clientSecret string) (*Client, error) {
	g := Client{TenantID: tenantID, ApplicationID: applicationID, ClientSecret: clientSecret}
	g.Lock()         // lock because we will refresh the token
	defer g.Unlock() // unlock after token refresh
	return &g, g.refreshToken()
}

// refreshToken refreshes the current Token. Grab's a new one and saves it within the Client instance
func (cli *Client) refreshToken() error {
	if cli.TenantID == "" {
		return fmt.Errorf("tenant ID is empty")
	}
	resource := fmt.Sprintf("/%v/oauth2/token", cli.TenantID)
	data := url.Values{}
	data.Add("grant_type", "client_credentials")
	data.Add("client_id", cli.ApplicationID)
	data.Add("client_secret", cli.ClientSecret)
	data.Add("resource", BaseURL)

	u, err := url.ParseRequestURI(LoginBaseURL)
	if err != nil {
		return fmt.Errorf("unable to parse URI: %v", err)
	}

	u.Path = resource
	req, err := http.NewRequest("POST", u.String(), bytes.NewBufferString(data.Encode()))

	if err != nil {
		return fmt.Errorf("HTTP Request Error: %v", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	var newToken Token
	err = cli.performRequest(req, &newToken) // perform the prepared request
	if err != nil {
		return fmt.Errorf("error on getting msgraph Token: %v", err)
	}
	cli.token = newToken
	return err
}

// makeGETAPICall performs an API-Call to the msgraph API. This func uses sync.Mutex to synchronize all API-calls
func (cli *Client) makeGETAPICall(apicall string, getParams url.Values, v interface{}) error {
	cli.Lock()
	defer cli.Unlock() // unlock when the func returns
	// Check token
	if cli.token.WantsToBeRefreshed() { // Token not valid anymore?
		err := cli.refreshToken()
		if err != nil {
			return err
		}
	}

	reqURL, err := url.ParseRequestURI(BaseURL)
	if err != nil {
		return fmt.Errorf("unable to parse URI %v: %v", BaseURL, err)
	}

	// Add Version to API-Call, the leading slash is always added by the calling func
	reqURL.Path = "/" + APIVersion + apicall

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return fmt.Errorf("HTTP request error: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", cli.token.GetAccessToken())

	if getParams == nil { // initialize getParams if it's nil
		getParams = url.Values{}
	}

	// TODO: Improve performance with using $skip & paging instead of retrieving all results with $top
	// TODO: MaxPageSize is currently 999, if there are any time more than 999 entries this will make the program unpredictable... hence start to use paging (!)
	getParams.Add("$top", strconv.Itoa(MaxPageSize))
	req.URL.RawQuery = getParams.Encode() // set query parameters

	return cli.performRequest(req, v)
}

// performRequest performs a pre-prepared http.Request and does the proper error-handling for it.
// does a json.Unmarshal into the v interface{} and returns the error of it if everything went well so far.
func (cli *Client) performRequest(req *http.Request, v interface{}) error {
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP response error: %v of http.Request: %v", err, req.URL)
	}
	defer resp.Body.Close() // close body when func returns

	body, err := ioutil.ReadAll(resp.Body) // read body first to append it to the error (if any)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Hint: this will mostly be the case if the tenant ID can not be found, the Application ID can not be found or the clientSecret is incorrect.
		// The cause will be described in the body, hence we have to return the body too for proper error-analysis
		return NewErr(resp.StatusCode, body)
	}

	//fmt.Println("Body: ", string(body))

	if err != nil {
		return fmt.Errorf("HTTP response read error: %v of http.Request: %v", err, req.URL)
	}

	return json.Unmarshal(body, &v) // return the error of the json unmarshal
}

// UnmarshalJSON implements the json unmarshal to be used by the json-library.
// This method additionally to loading the TenantID, ApplicationID and ClientSecret
// immediately gets a Token from msgraph (hence initialize this GraphAPI instance)
// and returns an error if any of the data provided is incorrect or the token can not be acquired
func (cli *Client) UnmarshalJSON(data []byte) error {
	tmp := struct {
		TenantID      string
		ApplicationID string
		ClientSecret  string
	}{}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	cli.TenantID = tmp.TenantID
	if cli.TenantID == "" {
		return fmt.Errorf("TenantID is empty")
	}
	cli.ApplicationID = tmp.ApplicationID
	if cli.ApplicationID == "" {
		return fmt.Errorf("ApplicationID is empty")
	}
	cli.ClientSecret = tmp.ClientSecret
	if cli.ClientSecret == "" {
		return fmt.Errorf("ClientSecret is empty")
	}

	// get a token and return the error (if any)
	err = cli.refreshToken()
	if err != nil {
		return fmt.Errorf("can't get Token: %v", err)
	}
	return nil
}
