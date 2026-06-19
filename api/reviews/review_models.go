package reviews

type GenerateReviewRequest struct {
	Rating int `json:"rating"`
}

type GenerateReviewResponse struct {
	Reviews []string `json:"reviews"`
}
