//go:build integration

// Repository tests that need a real Postgres. Run with:
//
//	GOOSE_DRIVER=postgres GOOSE_DBSTRING=... go test -tags=integration ./internal/repository/
//
// They truncate the tables they touch, so point GOOSE_DBSTRING at a scratch database.
package repository

import (
	"context"
	"os"
	"testing"

	"github.com/algoamigoo/micdrop/internal/db"
)

func testRepo(t *testing.T) *Repository {
	t.Helper()

	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		t.Skip("GOOSE_DBSTRING not set")
	}

	database, err := db.New(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	// Start from empty tables; ids keep advancing, which the assertions tolerate.
	for _, table := range []string{"prompt_votes", "response_votes", "responses", "prompts", "users"} {
		if _, err := database.Exec("TRUNCATE " + table + " RESTART IDENTITY CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
	return New(database)
}

func mustUser(t *testing.T, r *Repository, userID string) {
	t.Helper()
	if _, err := r.CreateUser(context.Background(), userID, "google-"+userID); err != nil {
		t.Fatalf("create user %s: %v", userID, err)
	}
}

func scores(t *testing.T, r *Repository, userID string) (prompt, response, total int) {
	t.Helper()
	user, err := r.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("get user %s: %v", userID, err)
	}
	return user.PromptScore, user.ResponseScore, user.TotalScore
}

func TestSoftDeletePrompt_KeepsKarma(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "author")
	mustUser(t, repo, "voter1")
	mustUser(t, repo, "voter2")

	prompt, err := repo.CreatePrompt(ctx, "author", "why did the chicken")
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	// Two upvotes plus the author's auto-vote: counter 3, karma 2.
	if _, err := repo.SetPromptVote(ctx, "voter1", prompt.PostID, VoteUp); err != nil {
		t.Fatalf("vote 1: %v", err)
	}
	if _, err := repo.SetPromptVote(ctx, "voter2", prompt.PostID, VoteUp); err != nil {
		t.Fatalf("vote 2: %v", err)
	}

	if p, r, total := scores(t, repo, "author"); p != 2 || r != 0 || total != 2 {
		t.Fatalf("before delete: prompt=%d response=%d total=%d, want 2/0/2", p, r, total)
	}

	if err := repo.DeletePrompt(ctx, prompt.PostID, "author"); err != nil {
		t.Fatalf("delete prompt: %v", err)
	}

	// Earned karma survives the delete.
	if p, _, total := scores(t, repo, "author"); p != 2 || total != 2 {
		t.Errorf("after delete: prompt=%d total=%d, want the earned 2/2", p, total)
	}

	if _, err := repo.GetPromptByID(ctx, prompt.PostID, "voter1"); err == nil {
		t.Error("expected GetPromptByID to 404 a deleted prompt")
	}

	fed, err := repo.ListPrompts(ctx, "newest", 10, 0, "voter1")
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if len(fed) != 0 {
		t.Errorf("expected the deleted prompt to be gone from the feed, got %d rows", len(fed))
	}

	// Voting a deleted prompt is a 404, not a silent counter bump.
	if _, err := repo.SetPromptVote(ctx, "voter1", prompt.PostID, VoteUp); err == nil {
		t.Error("expected voting a deleted prompt to fail")
	}
}

func TestDeletePrompt_NotAuthor(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "author")
	mustUser(t, repo, "mallory")

	prompt, err := repo.CreatePrompt(ctx, "author", "hello")
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	if err := repo.DeletePrompt(ctx, prompt.PostID, "mallory"); err != ErrNotAuthor {
		t.Errorf("expected ErrNotAuthor, got %v", err)
	}
	if _, err := repo.GetPromptByID(ctx, prompt.PostID, "author"); err != nil {
		t.Errorf("a rejected delete must leave the prompt intact, got %v", err)
	}
}

func TestSoftDeletePrompt_CascadesToResponses(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "author")
	mustUser(t, repo, "riff")

	prompt, err := repo.CreatePrompt(ctx, "author", "setup")
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	resp, err := repo.CreateResponse(ctx, prompt.PostID, "riff", "punchline")
	if err != nil {
		t.Fatalf("create response: %v", err)
	}
	if _, err := repo.SetResponseVote(ctx, "author", resp.ResponseID, VoteUp); err != nil {
		t.Fatalf("vote response: %v", err)
	}
	if _, _, total := scores(t, repo, "riff"); total != 1 {
		t.Fatalf("expected riff karma 1 from the author's upvote, got %d", total)
	}

	if err := repo.DeletePrompt(ctx, prompt.PostID, "author"); err != nil {
		t.Fatalf("delete prompt: %v", err)
	}

	if _, err := repo.GetResponseByID(ctx, resp.ResponseID, "riff"); err == nil {
		t.Error("expected the response to be soft-deleted with its prompt")
	}
	responses, err := repo.ListResponsesForPrompt(ctx, prompt.PostID, 10, 0, "riff")
	if err != nil {
		t.Fatalf("list responses: %v", err)
	}
	if len(responses) != 0 {
		t.Errorf("expected no responses, got %d", len(responses))
	}
	// The responder keeps the karma they earned: deleting someone else's prompt is
	// not their punishment.
	if _, r, total := scores(t, repo, "riff"); r != 1 || total != 1 {
		t.Errorf("expected the responder to keep karma 1, got %d/%d", r, total)
	}
}

func TestDeleteResponse_DecrementsCountButKeepsKarma(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "author")
	mustUser(t, repo, "riff")

	prompt, err := repo.CreatePrompt(ctx, "author", "setup")
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	resp, err := repo.CreateResponse(ctx, prompt.PostID, "riff", "punchline")
	if err != nil {
		t.Fatalf("create response: %v", err)
	}
	if _, err := repo.SetResponseVote(ctx, "author", resp.ResponseID, VoteUp); err != nil {
		t.Fatalf("vote: %v", err)
	}
	if _, _, total := scores(t, repo, "riff"); total != 1 {
		t.Fatalf("expected riff karma 1 from the author's upvote, got %d", total)
	}

	if err := repo.DeleteResponse(ctx, resp.ResponseID, "riff"); err != nil {
		t.Fatalf("delete response: %v", err)
	}

	after, err := repo.GetPromptByID(ctx, prompt.PostID, "riff")
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if after.ResponseCount != 0 {
		t.Errorf("expected response_count 0, got %d", after.ResponseCount)
	}
	if _, _, total := scores(t, repo, "riff"); total != 1 {
		t.Errorf("expected the earned karma to survive, got %d", total)
	}
}

func TestUpdatePrompt_OwnershipAndEditedFlag(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "author")
	mustUser(t, repo, "mallory")

	prompt, err := repo.CreatePrompt(ctx, "author", "typo here")
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	if prompt.Edited {
		t.Error("a fresh prompt must not be flagged as edited")
	}

	if _, err := repo.UpdatePrompt(ctx, prompt.PostID, "mallory", "hijacked"); err != ErrNotAuthor {
		t.Errorf("expected ErrNotAuthor, got %v", err)
	}

	updated, err := repo.UpdatePrompt(ctx, prompt.PostID, "author", "fixed")
	if err != nil {
		t.Fatalf("update prompt: %v", err)
	}
	if updated.Body != "fixed" {
		t.Errorf("expected body %q, got %q", "fixed", updated.Body)
	}
	if !updated.Edited {
		t.Error("expected the edited flag to be set after an update")
	}
	if updated.PromptUpvotes != 1 {
		t.Errorf("editing must not touch the vote counter, got %d", updated.PromptUpvotes)
	}

	// The flag has to survive a re-read, not just the UPDATE ... RETURNING row.
	fetched, err := repo.GetPromptByID(ctx, prompt.PostID, "author")
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if !fetched.Edited {
		t.Error("expected edited=true on a fresh read")
	}
}

func TestUpdateResponse_OwnershipAndEditedFlag(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "author")
	mustUser(t, repo, "riff")

	prompt, err := repo.CreatePrompt(ctx, "author", "setup")
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	resp, err := repo.CreateResponse(ctx, prompt.PostID, "riff", "typo")
	if err != nil {
		t.Fatalf("create response: %v", err)
	}
	if resp.Edited {
		t.Error("a fresh response must not be flagged as edited")
	}

	if _, err := repo.UpdateResponse(ctx, resp.ResponseID, "author", "hijacked"); err != ErrNotAuthor {
		t.Errorf("expected ErrNotAuthor, got %v", err)
	}

	updated, err := repo.UpdateResponse(ctx, resp.ResponseID, "riff", "fixed")
	if err != nil {
		t.Fatalf("update response: %v", err)
	}
	if updated.Body != "fixed" {
		t.Errorf("expected body %q, got %q", "fixed", updated.Body)
	}
	if !updated.Edited {
		t.Error("expected the edited flag to be set after an update")
	}
}

// A vote that flips direction must not double-count when karma is recomputed.
// Deleting is not a way to shed downvotes.
func TestDeletePrompt_KeepsDownvotePenalty(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "author")
	mustUser(t, repo, "voter")

	prompt, err := repo.CreatePrompt(ctx, "author", "setup")
	if err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	if _, err := repo.SetPromptVote(ctx, "voter", prompt.PostID, VoteDown); err != nil {
		t.Fatalf("downvote: %v", err)
	}
	if p, _, _ := scores(t, repo, "author"); p != -1 {
		t.Fatalf("expected karma -1, got %d", p)
	}

	if err := repo.DeletePrompt(ctx, prompt.PostID, "author"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if p, _, _ := scores(t, repo, "author"); p != -1 {
		t.Errorf("expected the downvote penalty to survive the delete, got %d", p)
	}
}

func TestFollow_IdempotentAndCounted(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "alice")
	mustUser(t, repo, "bob")

	if err := repo.Follow(ctx, "alice", "bob"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := repo.Follow(ctx, "alice", "bob"); err != nil {
		t.Errorf("following twice should be a no-op, got %v", err)
	}

	counts, err := repo.GetFollowCounts(ctx, "bob", "alice")
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if counts.FollowersCount != 1 || !counts.IsFollowing {
		t.Errorf("got %+v, want followers 1 and is_following", counts)
	}

	if err := repo.Unfollow(ctx, "alice", "bob"); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if err := repo.Unfollow(ctx, "alice", "bob"); err != nil {
		t.Errorf("unfollowing twice should be a no-op, got %v", err)
	}

	if counts, err = repo.GetFollowCounts(ctx, "bob", "alice"); err != nil {
		t.Fatalf("counts: %v", err)
	}
	if counts.FollowersCount != 0 || counts.IsFollowing {
		t.Errorf("got %+v, want cleared", counts)
	}
}

func TestFollow_SelfAndUnknown(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "alice")

	if err := repo.Follow(ctx, "alice", "alice"); err != ErrSelfFollow {
		t.Errorf("expected ErrSelfFollow, got %v", err)
	}
	if err := repo.Follow(ctx, "alice", "ghost"); err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// An anonymous viewer sees counts but never a follow state.
func TestGetFollowCounts_AnonymousViewer(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "alice")
	mustUser(t, repo, "bob")
	if err := repo.Follow(ctx, "alice", "bob"); err != nil {
		t.Fatalf("follow: %v", err)
	}

	counts, err := repo.GetFollowCounts(ctx, "bob", "")
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if counts.FollowersCount != 1 {
		t.Errorf("expected followers 1, got %d", counts.FollowersCount)
	}
	if counts.IsFollowing {
		t.Error("an anonymous viewer must not be following")
	}
}

// Both directions, pagination, and the effect of unfollowing.
func TestListFollowersAndFollowing(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t)

	mustUser(t, repo, "alice")
	mustUser(t, repo, "bob")
	mustUser(t, repo, "carol")

	if err := repo.Follow(ctx, "alice", "bob"); err != nil {
		t.Fatalf("alice follows bob: %v", err)
	}
	if err := repo.Follow(ctx, "carol", "bob"); err != nil {
		t.Fatalf("carol follows bob: %v", err)
	}
	if err := repo.Follow(ctx, "bob", "carol"); err != nil {
		t.Fatalf("bob follows carol: %v", err)
	}

	followers, err := repo.ListFollowers(ctx, "bob", 10, 0)
	if err != nil {
		t.Fatalf("followers: %v", err)
	}
	if len(followers) != 2 {
		t.Fatalf("expected 2 followers of bob, got %d", len(followers))
	}
	seen := map[string]bool{}
	for _, u := range followers {
		seen[u.UserID] = true
	}
	if !seen["alice"] || !seen["carol"] {
		t.Errorf("unexpected follower set: %+v", followers)
	}

	following, err := repo.ListFollowing(ctx, "bob", 10, 0)
	if err != nil {
		t.Fatalf("following: %v", err)
	}
	if len(following) != 1 || following[0].UserID != "carol" {
		t.Errorf("expected bob to follow only carol, got %+v", following)
	}

	// Pagination has to actually page.
	page, err := repo.ListFollowers(ctx, "bob", 1, 0)
	if err != nil {
		t.Fatalf("followers page: %v", err)
	}
	if len(page) != 1 {
		t.Errorf("expected the limit to apply, got %d", len(page))
	}
	next, err := repo.ListFollowers(ctx, "bob", 1, 1)
	if err != nil {
		t.Fatalf("followers page 2: %v", err)
	}
	if len(next) != 1 || next[0].UserID == page[0].UserID {
		t.Errorf("expected a different user on page 2, got %+v then %+v", page, next)
	}

	// Unfollowing removes them from both directions.
	if err := repo.Unfollow(ctx, "alice", "bob"); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if followers, err = repo.ListFollowers(ctx, "bob", 10, 0); err != nil {
		t.Fatalf("followers after unfollow: %v", err)
	}
	if len(followers) != 1 || followers[0].UserID != "carol" {
		t.Errorf("expected only carol to remain, got %+v", followers)
	}
}
