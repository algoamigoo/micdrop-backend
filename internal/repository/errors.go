package repository

import "errors"

var (
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

	// ErrNotAuthor is returned when a user tries to edit or delete content
	// authored by someone else.
	ErrNotAuthor = errors.New("only the author can modify this")

	// ErrSelfFollow is returned when a user tries to follow themselves.
	ErrSelfFollow = errors.New("you cannot follow yourself")
)
