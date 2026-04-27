package dto

type PlayGroundRequest struct {
	Model             string `json:"model,omitempty"`
	Group             string `json:"group,omitempty"`
	SpecificChannelID *int   `json:"specific_channel_id,omitempty"`
}
