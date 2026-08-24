package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"minidocker/internal/api/dto"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = os.Getenv("MINIDOCKER_HOST")
	}
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "unix", socketPath)
		},
	}

	return &Client{
		httpClient: &http.Client{Transport: transport, Timeout: 0},
		baseURL:    "http://unix/" + APIVersion,
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo contactar a minidockerd (¿está corriendo?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func decodeAPIError(resp *http.Response) error {
	var apiErr dto.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != "" {
		return fmt.Errorf("%s", apiErr.Error)
	}
	return fmt.Errorf("minidockerd respondió %d", resp.StatusCode)
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ping", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("no se pudo contactar a minidockerd: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}
	return nil
}

func (c *Client) CreateContainer(ctx context.Context, req dto.CreateContainerRequest) (*dto.CreateContainerResponse, error) {
	var out dto.CreateContainerResponse
	if err := c.doJSON(ctx, http.MethodPost, "/containers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) StartContainer(ctx context.Context, id string, req dto.StartContainerRequest) (*dto.StartContainerResponse, error) {
	var out dto.StartContainerResponse
	path := fmt.Sprintf("/containers/%s/start", id)
	if err := c.doJSON(ctx, http.MethodPost, path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) StopContainer(ctx context.Context, id string) error {
	var out dto.StopContainerResponse
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/containers/%s/stop", id), nil, &out)
}

func (c *Client) DeleteContainer(ctx context.Context, id string) error {
	var out dto.DeleteContainerResponse
	return c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("/containers/%s", id), nil, &out)
}

func (c *Client) ListContainers(ctx context.Context) (*dto.ListContainersResponse, error) {
	var out dto.ListContainersResponse
	if err := c.doJSON(ctx, http.MethodGet, "/containers", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) InspectContainer(ctx context.Context, id string) (*dto.ContainerView, error) {
	var out dto.ContainerView
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/containers/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) StreamLogs(ctx context.Context, id string, follow bool, w io.Writer) error {
	path := fmt.Sprintf("/containers/%s/logs", id)
	if follow {
		path += "?follow=true"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func (c *Client) Stats(ctx context.Context, id string) (*dto.StatsResponse, error) {
	var out dto.StatsResponse
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/containers/%s/stats", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
