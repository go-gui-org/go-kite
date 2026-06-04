package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	pdsHost          = "https://bsky.social/xrpc"
	apiTimelineLimit = 50
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func createSession(identifier, password string) (BSkySession, error) {
	payload, err := json.Marshal(bSkyCreateSessionRequest{
		Identifier: identifier,
		Password:   password,
	})
	if err != nil {
		return BSkySession{}, err
	}

	req, err := http.NewRequest(http.MethodPost,
		pdsHost+"/com.atproto.server.createSession",
		bytes.NewReader(payload),
	)
	if err != nil {
		return BSkySession{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return BSkySession{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return BSkySession{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return BSkySession{}, fmt.Errorf("%s", resp.Status)
	}

	var session BSkySession
	if err := json.Unmarshal(body, &session); err != nil {
		return BSkySession{}, err
	}
	return session, nil
}

func refreshBSkySession(session BSkySession) (refreshSessionResponse, error) {
	req, err := http.NewRequest(http.MethodPost,
		pdsHost+"/com.atproto.server.refreshSession", nil)
	if err != nil {
		return refreshSessionResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+session.RefreshJWT)

	resp, err := httpClient.Do(req)
	if err != nil {
		return refreshSessionResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return refreshSessionResponse{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return refreshSessionResponse{}, fmt.Errorf("%s", resp.Status)
	}

	var refresh refreshSessionResponse
	if err := json.Unmarshal(body, &refresh); err != nil {
		return refreshSessionResponse{}, err
	}
	return refresh, nil
}

func getTimeline(session BSkySession) (bSkyTimeline, error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/app.bsky.feed.getTimeline?limit=%d", pdsHost, apiTimelineLimit), nil,
	)
	if err != nil {
		return bSkyTimeline{}, err
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJWT)

	resp, err := httpClient.Do(req)
	if err != nil {
		return bSkyTimeline{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return bSkyTimeline{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return bSkyTimeline{}, fmt.Errorf("%s", resp.Status)
	}

	var timeline bSkyTimeline
	if err := json.Unmarshal(body, &timeline); err != nil {
		return bSkyTimeline{}, err
	}
	return timeline, nil
}

func getBlob(did, cid string) ([]byte, error) {
	encodedDID := url.QueryEscape(did)
	encodedCID := url.QueryEscape(cid)
	endpoint := fmt.Sprintf("%s/com.atproto.sync.getBlob?did=%s&cid=%s",
		pdsHost, encodedDID, encodedCID)

	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	return body, nil
}
