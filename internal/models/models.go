package models

import "time"

type User struct {
	UserID        string    `json:"user_id"        db:"user_id"`
	UserName      string    `json:"user_name"      db:"user_name"`
	PromptScore   int       `json:"prompt_score"   db:"prompt_score"`
	ResponseScore int       `json:"response_score" db:"response_score"`
	TotalScore    int       `json:"total_score"    db:"total_score"`
	CreatedAt     time.Time `json:"created_at"     db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"     db:"updated_at"`
}

type Prompt struct {
	PostID        int64     `json:"post_id"        db:"post_id"`
	UserID        string    `json:"user_id"        db:"user_id"`
	UserName      string    `json:"user_name"      db:"user_name"`
	Body          string    `json:"body"           db:"body"`
	PromptUpvotes int       `json:"prompt_upvotes" db:"prompt_upvotes"`
	ResponseCount int       `json:"response_count" db:"response_count"`
	CreatedAt     time.Time `json:"created_at"     db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"     db:"updated_at"`
}

type Response struct {
	ResponseID      int64     `json:"response_id"      db:"response_id"`
	PostID          int64     `json:"post_id"          db:"post_id"`
	UserID          string    `json:"user_id"          db:"user_id"`
	UserName        string    `json:"user_name"        db:"user_name"`
	Body            string    `json:"body"             db:"body"`
	ResponseUpvotes int       `json:"response_upvotes" db:"response_upvotes"`
	CreatedAt       time.Time `json:"created_at"       db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"       db:"updated_at"`
}
