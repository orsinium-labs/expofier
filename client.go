package expofier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	Client     http.Client
	BaseURL    string
	AcessToken string
}

type Ticket string

type Resp struct {
	Ticket Ticket
	Error  error
}

func Flatten(resps []Resp) ([]Ticket, error) {
	tickets := make([]Ticket, 0, len(resps))
	for _, resp := range resps {
		if resp.Error != nil {
			return nil, resp.Error
		}
		tickets = append(tickets, resp.Ticket)
	}
	return tickets, nil
}

type sendResp struct {
	Data   []messageResponse `json:"data"`
	Errors []any             `json:"errors"`
}

type receiptsResp struct {
	Data   map[Ticket]messageResponse `json:"data"`
	Errors []any                      `json:"errors"`
}

type messageResponse struct {
	Status  string `json:"status"`
	ID      Ticket `json:"id"`
	Message string `json:"message"`
	Details struct {
		Error string `json:"error"`
	} `json:"details"`
}

func (c Client) SendMessage(ctx context.Context, msg Message) (Ticket, error) {
	resps, err := c.SendMessages(ctx, []Message{msg})
	if err != nil {
		return "", err
	}
	resp := resps[0]
	return resp.Ticket, resp.Error
}

func (c Client) SendMessages(ctx context.Context, msgs []Message) ([]Resp, error) {
	if len(msgs) > 100 {
		return nil, ErrTooManyMessages
	}

	// Send request.
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://exp.host"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/--/api/v2/push/send"
	raw, err := json.Marshal(msgs)
	if err != nil {
		return nil, fmt.Errorf("serialize messages: %w", err)
	}
	body := bytes.NewReader(raw)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	if c.AcessToken != "" {
		req.Header.Add("Authorization", "Bearer "+c.AcessToken)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Handle response.
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrTooManyRequests
	}
	if resp.StatusCode >= 500 {
		return nil, ErrServerError
	}
	var r *sendResp
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(r.Errors) > 0 {
		return nil, ErrBadRequest
	}
	if len(r.Data) != len(msgs) {
		return nil, ErrInvalidTicketCount
	}
	resps := make([]Resp, 0, len(msgs))
	for _, rawResp := range r.Data {
		resp := Resp{Ticket: rawResp.ID}
		if rawResp.Status != "ok" {
			resp.Error = parseErr(rawResp)
		}
		resps = append(resps, resp)
	}
	return resps, nil
}

type Receipt struct {
	Error error
}

func (c Client) FetchReceipt(ctx context.Context, ticket Ticket) *Receipt {
	rs, err := c.FetchReceipts(ctx, []Ticket{ticket})
	if err != nil {
		return &Receipt{Error: err}
	}
	r, found := rs[ticket]
	if !found {
		return nil
	}
	return &r
}

func (c Client) FetchReceipts(ctx context.Context, tickets []Ticket) (map[Ticket]Receipt, error) {
	if len(tickets) > 300 {
		return nil, ErrTooManyTickets
	}

	// Send request.
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://exp.host"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/--/api/v2/push/getReceipts"
	raw, err := json.Marshal(struct {
		IDs []Ticket `json:"ids"`
	}{tickets})
	if err != nil {
		return nil, fmt.Errorf("serialize tickets: %w", err)
	}
	body := bytes.NewReader(raw)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	if c.AcessToken != "" {
		req.Header.Add("Authorization", "Bearer "+c.AcessToken)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	// Handle response
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrTooManyRequests
	}
	if resp.StatusCode >= 500 {
		return nil, ErrServerError
	}
	var r *receiptsResp
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(r.Errors) > 0 {
		return nil, ErrBadRequest
	}
	receipts := make(map[Ticket]Receipt, len(r.Data))
	for ticket, rawResp := range r.Data {
		receipt := Receipt{}
		if rawResp.Status != "ok" {
			receipt.Error = parseErr(rawResp)
		}
		receipts[ticket] = receipt
	}
	return receipts, nil
}

func repeat(msgs []Message, err error) []Resp {
	resp := Resp{Error: err}
	resps := make([]Resp, 0, len(msgs))
	for range len(msgs) {
		resps = append(resps, resp)
	}
	return resps
}

func parseErr(r messageResponse) error {
	switch r.Details.Error {
	case "DeviceNotRegistered":
		return ErrDeviceNotRegistered
	case "MessageTooBig":
		return ErrMessageTooBig
	case "MessageRateExceeded":
		return ErrMessageRateExceeded
	case "MismatchSenderId":
		return ErrMismatchSenderID
	case "InvalidCredentials":
		return ErrInvalidCredentials
	default:
		if r.Message == "" {
			return ErrUnknown
		}
		return errors.New(r.Message)
	}
}
