package entrystore

type Entry struct {
	UserID              string   `json:"user_id"`
	Date                string   `json:"date"`
	Notes               string   `json:"notes,omitempty"`
	Tags                []string `json:"tags,omitempty"`
	Mood                string   `json:"mood,omitempty"`
	DisciplineScore     int      `json:"discipline_score,omitempty"`
	ScreenshotFileIDs   []string `json:"screenshot_file_ids,omitempty"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}
