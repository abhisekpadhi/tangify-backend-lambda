package reviews

type GenerateReviewRequest struct {
	Rating int `json:"rating"`
}

type GenerateReviewResponse struct {
	Review string `json:"review"`
}
