package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const UserAgentURL = "https://raw.githubusercontent.com/jnrbsn/user-agents/refs/heads/main/user-agents.json"

func LoadUserAgent(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, UserAgentURL, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %s", ErrUpstream, res.Status)
	}

	var agents []string
	if err := json.NewDecoder(res.Body).Decode(&agents); err != nil {
		return "", err
	}

	return agents[0], nil
}
