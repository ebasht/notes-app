package notes

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxTitleLen = 200
	maxBodyLen  = 20000
)

type Note struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Store interface {
	List(ctx context.Context) ([]Note, error)
	Create(ctx context.Context, title, body string) (Note, error)
	Update(ctx context.Context, id int64, title, body string) (Note, error)
	Delete(ctx context.Context, id int64) error
}

var ErrNotFound = errors.New("заметка не найдена")

type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

func Normalize(title, body string) (string, string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", "", &ValidationError{Msg: "Укажите заголовок"}
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return "", "", &ValidationError{Msg: "Заголовок длиннее 200 символов"}
	}
	if utf8.RuneCountInString(body) > maxBodyLen {
		return "", "", &ValidationError{Msg: "Текст длиннее 20000 символов"}
	}
	return title, body, nil
}
