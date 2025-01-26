package rewind

import (
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"
)

type RequestWriteDB struct {
	ID          ulid.ULID `json:"id"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	Headers     []byte    `json:"headers"`
	Cookies     []byte    `json:"cookies"`
	Body        []byte    `json:"body"`
	QueryParams []byte    `json:"query_params"`
	RecordedAt  time.Time `json:"recorded_at"`
}

type App struct {
	DB     *sql.DB
	Config *Config
}
