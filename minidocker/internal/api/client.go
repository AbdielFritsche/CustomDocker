package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(socketPath string) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}

// GetContainers consulta la lista de contenedores al daemon vía Unix socket
func (c *Client) GetContainers() ([]*ContainerResponse, error) {
	resp, err := c.httpClient.Get("http://unix/containers/list")
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar con minidockerd: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("el daemon respondió con error status: %d", resp.StatusCode)
	}

	var containers []*ContainerResponse
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("error deserializando respuesta del daemon: %w", err)
	}

	return containers, nil
}

func (c *Client) StopContainer(idOrName string) error {
	url := fmt.Sprintf("http://unix/containers/stop?id=%s", idOrName)
	resp, err := c.httpClient.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("error enviando stop al daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fallo al detener: el daemon respondió con código %d", resp.StatusCode)
	}
	return nil
}

// StartContainer solicita al daemon iniciar un contenedor
func (c *Client) StartContainer(idOrName string, overrideCmd ...string) error {
	url := fmt.Sprintf("http://unix/containers/start?id=%s", idOrName)

	var reqBody []byte
	if len(overrideCmd) > 0 {
		reqBody, _ = json.Marshal(map[string][]string{"command": overrideCmd})
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error enviando start al daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fallo al iniciar: el daemon respondió con código %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) DaemonStatus() (*StatusResponse, error) {
	resp, err := c.httpClient.Get("http://unix/health")
	if err != nil {
		return nil, fmt.Errorf("daemon no responde en socket: %w", err)
	}
	defer resp.Body.Close()

	var status StatusResponse
	err = json.NewDecoder(resp.Body).Decode(&status)
	return &status, err
}

func (c *Client) Create(req CreateRequest) (string, error) {
	data, _ := json.Marshal(req)
	resp, err := c.httpClient.Post("http://unix/containers/create", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res ContainerResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.ID, nil
}

func (c *Client) Start(id string) error {
	resp, err := c.httpClient.Post(fmt.Sprintf("http://unix/containers/start?id=%s", id), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
