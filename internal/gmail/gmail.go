package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	authURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL = "https://oauth2.googleapis.com/token"
	apiBase  = "https://gmail.googleapis.com/gmail/v1/users/me"
	scope    = "https://www.googleapis.com/auth/gmail.readonly"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

type Client struct {
	clientID     string
	clientSecret string
	accessToken  string
	refreshToken string
	http         *http.Client
}

func NewClient(clientID, clientSecret, accessToken, refreshToken string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		accessToken:  accessToken,
		refreshToken: refreshToken,
		http:         &http.Client{Timeout: 30 * time.Second},
	}
}

const redirectPort = "8585"
const redirectURI = "http://localhost:" + redirectPort

func AuthURL(clientID string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", scope)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	return authURL + "?" + params.Encode()
}

// WaitForCode starts a local server and waits for Google to redirect with the auth code.
func WaitForCode() (string, error) {
	ln, err := net.Listen("tcp", ":"+redirectPort)
	if err != nil {
		return "", fmt.Errorf("cannot start local server on port %s: %w", redirectPort, err)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	srv := &http.Server{}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback: %s", r.URL.RawQuery)
			fmt.Fprintln(w, "Error: no code received. Close this tab.")
			return
		}
		fmt.Fprintln(w, "<html><body><h2>Herald authorized!</h2><p>You can close this tab.</p></body></html>")
		codeCh <- code
	})

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case code := <-codeCh:
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		return code, nil
	case err := <-errCh:
		return "", err
	case <-time.After(2 * time.Minute):
		return "", fmt.Errorf("timeout waiting for authorization")
	}
}

func ExchangeCode(clientID, clientSecret, code string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("code", code)
	params.Set("client_id", clientID)
	params.Set("client_secret", clientSecret)
	params.Set("redirect_uri", redirectURI)
	params.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(tokenURL, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var tok TokenResponse
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	if tok.Error != "" {
		return nil, fmt.Errorf("%s: %s", tok.Error, tok.ErrorDesc)
	}
	return &tok, nil
}

func (c *Client) refreshAccessToken() error {
	params := url.Values{}
	params.Set("refresh_token", c.refreshToken)
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("grant_type", "refresh_token")

	resp, err := http.PostForm(tokenURL, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var tok TokenResponse
	if err := json.Unmarshal(data, &tok); err != nil {
		return err
	}
	if tok.Error != "" {
		return fmt.Errorf("%s: %s", tok.Error, tok.ErrorDesc)
	}
	c.accessToken = tok.AccessToken
	return nil
}

type Email struct {
	Subject string
	From    string
	Date    string
	Body    string
}

type messageList struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

type messageDetail struct {
	Payload struct {
		Headers  []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
		Parts    []mimePart `json:"parts"`
		Body     struct{ Data string `json:"data"` } `json:"body"`
		MimeType string     `json:"mimeType"`
	} `json:"payload"`
}

type mimePart struct {
	MimeType string     `json:"mimeType"`
	Body     struct{ Data string `json:"data"` } `json:"body"`
	Parts    []mimePart `json:"parts"`
}

func (c *Client) FetchEmails(senderEmails []string, sinceDate string) ([]Email, error) {
	var froms []string
	for _, e := range senderEmails {
		froms = append(froms, "from:"+e)
	}
	afterDate := strings.ReplaceAll(sinceDate, "-", "/")
	q := "(" + strings.Join(froms, " OR ") + ") after:" + afterDate

	data, err := c.get(fmt.Sprintf("%s/messages?maxResults=50&q=%s", apiBase, url.QueryEscape(q)))
	if err != nil {
		return nil, err
	}

	var list messageList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	var emails []Email
	for _, msg := range list.Messages {
		e, err := c.fetchMessage(msg.ID)
		if err != nil {
			continue
		}
		emails = append(emails, *e)
	}
	return emails, nil
}

func (c *Client) fetchMessage(id string) (*Email, error) {
	data, err := c.get(fmt.Sprintf("%s/messages/%s?format=full", apiBase, id))
	if err != nil {
		return nil, err
	}
	var msg messageDetail
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	e := &Email{}
	for _, h := range msg.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "subject":
			e.Subject = h.Value
		case "from":
			e.From = h.Value
		case "date":
			e.Date = h.Value
		}
	}
	e.Body = extractBody(msg.Payload.Parts, msg.Payload.MimeType, msg.Payload.Body.Data)
	if len(e.Body) > 8000 {
		e.Body = e.Body[:8000]
	}
	return e, nil
}

func extractBody(parts []mimePart, mimeType, directData string) string {
	for _, p := range parts {
		if p.MimeType == "text/plain" && p.Body.Data != "" {
			return decodeBase64(p.Body.Data)
		}
	}
	for _, p := range parts {
		if strings.HasPrefix(p.MimeType, "multipart/") {
			if text := extractBody(p.Parts, p.MimeType, ""); text != "" {
				return text
			}
		}
	}
	for _, p := range parts {
		if p.MimeType == "text/html" && p.Body.Data != "" {
			return stripHTML(decodeBase64(p.Body.Data))
		}
	}
	if directData != "" {
		text := decodeBase64(directData)
		if mimeType == "text/html" {
			return stripHTML(text)
		}
		return text
	}
	return ""
}

func decodeBase64(s string) string {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		data, err = base64.StdEncoding.DecodeString(s)
		if err != nil {
			return ""
		}
	}
	return string(data)
}

var (
	reHTML     = regexp.MustCompile(`<[^>]+>`)
	reSpaces   = regexp.MustCompile(`[ \t]{2,}`)
	reNewlines = regexp.MustCompile(`\n{3,}`)
)

func stripHTML(s string) string {
	s = reHTML.ReplaceAllString(s, " ")
	s = reSpaces.ReplaceAllString(s, " ")
	s = reNewlines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func (c *Client) get(endpoint string) ([]byte, error) {
	data, statusCode, err := c.doGet(endpoint)
	if err != nil {
		return nil, err
	}
	if statusCode == 401 {
		if err := c.refreshAccessToken(); err != nil {
			return nil, fmt.Errorf("token refresh: %w", err)
		}
		data, statusCode, err = c.doGet(endpoint)
		if err != nil {
			return nil, err
		}
	}
	if statusCode >= 400 {
		var apiErr struct {
			Error struct{ Message string `json:"message"` } `json:"error"`
		}
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("Gmail API: %s", apiErr.Error.Message)
		}
		return nil, fmt.Errorf("Gmail API status %d", statusCode)
	}
	return data, nil
}

func (c *Client) doGet(endpoint string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}

func FormatForPrompt(emails []Email) string {
	var b bytes.Buffer
	for i, e := range emails {
		fmt.Fprintf(&b, "--- EMAIL %d ---\nFrom: %s\nSubject: %s\nDate: %s\n\n%s\n\n", i+1, e.From, e.Subject, e.Date, e.Body)
	}
	return b.String()
}
