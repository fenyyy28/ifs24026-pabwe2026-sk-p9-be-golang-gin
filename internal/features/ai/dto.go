package ai

// Source adalah satu sumber rujukan hasil penelusuran Search Engine.
type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// GenerateRequest adalah payload pembuatan draf artikel via AI.
type GenerateRequest struct {
	Topic            string `json:"topic" binding:"required"`
	Category         string `json:"category"`
	IncludeWebSearch bool   `json:"include_web_search"`
}

// GenerateResponse adalah draf artikel utuh beserta sumber rujukan web.
type GenerateResponse struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Excerpt  string   `json:"excerpt"`
	Content  string   `json:"content"`
	Sources  []Source `json:"sources"`
}

// generated adalah bentuk JSON yang diminta dari model AI.
type generated struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Excerpt  string `json:"excerpt"`
	Content  string `json:"content"`
}
