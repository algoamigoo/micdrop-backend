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

	// ErrUserIDTaken is returned when the requested user_id (username) already exists.
	ErrUserIDTaken = errors.New("that username is already taken")

	// ErrGoogleIDTaken is returned when the Google account is already linked to a user.
	ErrGoogleIDTaken = errors.New("this google account is already registered")

	// ErrInvalidLink is returned when a profile link has a bad type or URL.
	ErrInvalidLink = errors.New("invalid profile link")
)
