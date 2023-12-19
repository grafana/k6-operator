package types

import "encoding/json"

// k6 REST API types.
// TODO: refactor with existing definitions in k6 api/v1?

type StatusAPIRequest struct {
	Data StatusAPIRequestData `json:"data"`
}

type StatusAPIRequestData struct {
	Attributes StatusAPIRequestDataAttributes `json:"attributes"`
	ID         string                         `json:"id"`
	Type       string                         `json:"type"`
}

type StatusAPIRequestDataAttributes struct {
	Paused  bool `json:"paused"`
	Stopped bool `json:"stopped"`
}

type SetupData struct {
	Data setUpData `json:"data"`
}

type setUpData struct {
	Type       string                  `json:"type"`
	ID         string                  `json:"id"`
	Attributes setupResponseAttributes `json:"attributes"`
}

type setupResponseAttributes struct {
	Data json.RawMessage `json:"data"`
}
