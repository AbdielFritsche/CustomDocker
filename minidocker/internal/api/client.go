package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
)

const baseURL = "http://minidockerd"

type Client struct {
	http *http.Client
}

func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	return &Client{
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", socketPath)
				},
			},
		},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo contactar a minidockerd (¿está corriendo? ¿existe /var/run/minidocker.sock?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr ErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Error != "" {
			return fmt.Errorf("%s", apiErr.Error)
		}
		return fmt.Errorf("minidockerd respondió %d", resp.StatusCode)
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	var resp CreateResponse
	if err := c.do(ctx, http.MethodPost, "/containers", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Start(ctx context.Context, idOrName string, override []string) error {
	return c.do(ctx, http.MethodPost, "/containers/"+idOrName+"/start", StartRequest{CommandOverride: override}, nil)
}

func (c *Client) Stop(ctx context.Context, idOrName string) error {
	return c.do(ctx, http.MethodPost, "/containers/"+idOrName+"/stop", nil, nil)
}

func (c *Client) Delete(ctx context.Context, idOrName string) error {
	return c.do(ctx, http.MethodDelete, "/containers/"+idOrName, nil, nil)
}

func (c *Client) Ps(ctx context.Context) (*PsResponse, error) {
	var resp PsResponse
	if err := c.do(ctx, http.MethodGet, "/containers", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Stats(ctx context.Context, idOrName string) (*StatsDTO, error) {
	var resp StatsDTO
	if err := c.do(ctx, http.MethodGet, "/containers/"+idOrName+"/stats", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
