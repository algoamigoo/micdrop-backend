package repository

import "errors"

var (
	// ErrAlreadyVoted is returned when a user tries to vote on an item they have already voted on.
	ErrAlreadyVoted = errors.New("user has already voted on this item")

	// ErrSelfVote is returned when a user tries to vote on their own prompt or response.
	ErrSelfVote = errors.New("user cannot vote on their own item")

	// ErrPromptNotFound is returned when a requested prompt does not exist.
	ErrPromptNotFound = errors.New("prompt not found")

	// ErrResponseNotFound is returned when a requested response does not exist.
	ErrResponseNotFound = errors.New("response not found")

	// ErrUserNotFound is returned when a requested user does not exist.
	ErrUserNotFound = errors.New("user not found")
)
