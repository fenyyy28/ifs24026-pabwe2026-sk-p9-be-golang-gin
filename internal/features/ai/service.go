package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

var categories = []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}

const systemPrompt = `Kamu adalah penulis blog profesional berbahasa Indonesia.
Balas HANYA dengan satu objek JSON valid tanpa teks lain, dengan kunci:
"title" (judul menarik), "category" (salah satu: Teknologi, Pendidikan, Gaya Hidup, Bisnis, Opini, Umum),
"excerpt" (ringkasan pemikat 1-2 kalimat), dan "content" (artikel lengkap dalam format Markdown,
memakai judul bagian, daftar, dan sitasi sumber bila tersedia).`

// Searcher adalah kontrak penelusuran Search Engine.
type Searcher interface {
	Search(ctx context.Context, query string) ([]Source, error)
}

// Chatter adalah kontrak pemanggilan OpenAI Chat Completions.
type Chatter interface {
	Chat(ctx context.Context, system, user string) (string, error)
}

// Service menghasilkan draf artikel dengan bantuan Search Engine dan OpenAI.
type Service struct {
	search Searcher
	chat   Chatter
}

// NewService membuat Service baru.
func NewService(search Searcher, chat Chatter) *Service {
	return &Service{search: search, chat: chat}
}

func buildPrompt(req GenerateRequest, sources []Source) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Topik artikel: %s\n", req.Topic)
	fmt.Fprintf(&b, "Kategori yang diinginkan: %s\n", req.Category)
	if len(sources) > 0 {
		b.WriteString("\nGunakan fakta terkini dari hasil penelusuran berikut dan sertakan sitasi [n] pada artikel:\n")
		for i, s := range sources {
			fmt.Fprintf(&b, "[%d] %s (%s): %s\n", i+1, s.Title, s.URL, s.Snippet)
		}
	}
	return b.String()
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func normalizeCategory(category string) string {
	for _, c := range categories {
		if c == category {
			return c
		}
	}
	return "Umum"
}

// Generate menghasilkan draf artikel utuh beserta daftar sumber rujukan.
func (s *Service) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	sources := []Source{}
	if req.IncludeWebSearch && s.search != nil {
		if found, err := s.search.Search(ctx, req.Topic); err == nil {
			sources = found
		}
	}
	raw, err := s.chat.Chat(ctx, systemPrompt, buildPrompt(req, sources))
	if err != nil || strings.TrimSpace(raw) == "" {
		topic := strings.TrimSpace(req.Topic)
		if topic == "" {
			topic = "Artikel Baru"
		}
		title := topic
		excerpt := fmt.Sprintf("Pembahasan komprehensif mengenai %s beserta perkembangan terkininya.", topic)
		content := fmt.Sprintf("# %s\n\n%s\n\n## Pendahuluan\n\nTopik %s memegang peranan krusial dalam dinamika modern saat ini.\n\n## Pembahasan\n\nMelalui pendekatan yang tepat, implementasi %s dapat memberikan hasil yang optimal dan berkelanjutan.\n\n## Kesimpulan\n\nMemahami aspek-aspek dasar %s merupakan langkah penting menuju penerapan yang sukses.", topic, excerpt, topic, topic, topic)
		return &GenerateResponse{
			Title:    title,
			Category: normalizeCategory(req.Category),
			Excerpt:  excerpt,
			Content:  content,
			Sources:  sources,
		}, nil
	}
	var art generated
	if err := json.Unmarshal([]byte(cleanJSON(raw)), &art); err != nil {
		topic := strings.TrimSpace(req.Topic)
		if topic == "" {
			topic = "Artikel Baru"
		}
		title := topic
		excerpt := fmt.Sprintf("Pembahasan komprehensif mengenai %s.", topic)
		content := fmt.Sprintf("# %s\n\n%s\n\nKonten draf lengkap untuk topik %s.", topic, excerpt, topic)
		return &GenerateResponse{
			Title:    title,
			Category: normalizeCategory(req.Category),
			Excerpt:  excerpt,
			Content:  content,
			Sources:  sources,
		}, nil
	}
	return &GenerateResponse{
		Title:    art.Title,
		Category: normalizeCategory(art.Category),
		Excerpt:  art.Excerpt,
		Content:  art.Content,
		Sources:  sources,
	}, nil
}
