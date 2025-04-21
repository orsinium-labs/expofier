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

type response struct {
	Data   []messageResponse `json:"data"`
	Errors []any             `json:"errors"`
}

type messageResponse struct {
	Status  string `json:"status"`
	ID      Ticket `json:"id"`
	Message string `json:"message"`
	Details struct {
		Error string `json:"error"`
	} `json:"details"`
}

func (c Client) Send(ctx context.Context, msg Message) (Ticket, error) {
	resp := c.SendMany(ctx, []Message{msg})[0]
	return resp.Ticket, resp.Error
}

func (c Client) SendMany(ctx context.Context, msgs []Message) []Resp {
	if len(msgs) > 100 {
		return repeat(msgs, ErrTooManyMessages)
	}
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://exp.host"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	url := baseURL + "/--/api/v2/push/send"
	raw, err := json.Marshal(msgs)
	if err != nil {
		return repeat(msgs, fmt.Errorf("serialize messages: %w", err))
	}
	body := bytes.NewReader(raw)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return repeat(msgs, fmt.Errorf("create request: %w", err))
	}
	req.Header.Add("Content-Type", "application/json")
	if c.AcessToken != "" {
		req.Header.Add("Authorization", "Bearer "+c.AcessToken)
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return repeat(msgs, fmt.Errorf("send request: %w", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return repeat(msgs, ErrTooManyRequests)
	}
	if resp.StatusCode >= 500 {
		return repeat(msgs, ErrServerError)
	}
	var r *response
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return repeat(msgs, fmt.Errorf("decode response: %w", err))
	}
	if len(r.Errors) > 0 {
		return repeat(msgs, ErrBadRequest)
	}
	if len(r.Data) != len(msgs) {
		return repeat(msgs, ErrInvalidTicketCount)
	}
	resps := make([]Resp, 0, len(msgs))
	for _, rawResp := range r.Data {
		resp := Resp{Ticket: rawResp.ID}
		if rawResp.Status != "ok" {
			resp.Error = parseErr(rawResp)
		}
		resps = append(resps, resp)
	}
	return resps
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
