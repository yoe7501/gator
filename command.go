package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/yoe7501/gator/internal/database"
)

type Command struct {
	name string
	args []string
}

type Commands struct {
	commandToHandler map[string]func(*state, Command) error
}

func (c *Commands) register(name string, f func(*state, Command) error) {
	c.commandToHandler[name] = f
}

func (c *Commands) run(s *state, cmd Command) error {
	handler, ok := c.commandToHandler[cmd.name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return handler(s, cmd)
}

func handlerLogin(s *state, cmd Command) error {
	if len(cmd.args) == 0 {
		return errors.New("username is required")
	}
	user, err := s.queries.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}
	err = s.cfg.SetUser(user.Name)
	if err != nil {
		return err
	}
	fmt.Println("User Set!")
	return nil
}

func handlerRegister(s *state, cmd Command) error {
	if len(cmd.args) == 0 {
		return errors.New("name required")
	}
	t := time.Now()
	user := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: t,
		UpdatedAt: t,
		Name:      cmd.args[0],
	}
	response, err := s.queries.CreateUser(context.Background(), user)
	if err != nil {
		return err
	}
	err = s.cfg.SetUser(response.Name)
	if err != nil {
		return err
	}
	fmt.Println("User registered!", response)
	return nil
}

func handlerReset(s *state, cmd Command) error {
	err := s.queries.DeleteAllUsers(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Success, removed all users")
	return nil
}

func handlerUsers(s *state, cmd Command) error {
	users, err := s.queries.GetUsers(context.Background())
	if err != nil {
		return err
	}
	for _, i := range users {
		a := ""
		if i.Name == s.cfg.UserName {
			a = " (current)"
		}
		fmt.Println("* " + i.Name + a)
	}
	return nil
}

func handlerAgg(s *state, cmd Command) error {
	if len(cmd.args) != 1 {
		fmt.Println("The Zen of Proverbs Optimize for simplicity")
		return nil
	}
	duration, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}
	fmt.Println("Collecting feeds every " + cmd.args[0])
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	for ; ; <-ticker.C {
		err = scrapeFeeds(s)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func handlerAddFeed(s *state, cmd Command, user database.User) error {
	if len(cmd.args) != 2 {
		return errors.New("addfeed <name> <url>")
	}
	t := time.Now()
	feed := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: t,
		UpdatedAt: t,
		Name:      cmd.args[0],
		Url:       cmd.args[1],
		UserID:    user.ID,
	}
	res, err := s.queries.CreateFeed(context.Background(), feed)
	if err != nil {
		return err
	}
	feedFollow := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: t,
		UpdatedAt: t,
		UserID:    user.ID,
		FeedID:    res.ID,
	}
	_, err = s.queries.CreateFeedFollow(context.Background(), feedFollow)
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}

func handlerFeeds(s *state, cmd Command) error {
	feeds, err := s.queries.GetFeeds(context.Background())
	if err != nil {
		return err
	}
	for _, f := range feeds {
		fmt.Println(f.Name)
		fmt.Println(f.Url)
		fmt.Println(f.UserName)

	}
	return nil
}

func handlerFollow(s *state, cmd Command, user database.User) error {
	if len(cmd.args) != 1 {
		return errors.New("follow <url>")
	}
	feed, err := s.queries.GetFeed(context.Background(), cmd.args[0])
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("no feed with that URL")
	}
	if err != nil {
		return err
	}
	t := time.Now()
	feedFollow := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: t,
		UpdatedAt: t,
		UserID:    user.ID,
		FeedID:    feed.ID,
	}
	response, err := s.queries.CreateFeedFollow(context.Background(), feedFollow)
	if err != nil {
		return err
	}
	fmt.Println(response.FeedName, response.UserName)
	return nil
}

func handlerFollowing(s *state, cmd Command, user database.User) error {
	feeds, err := s.queries.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}
	if len(feeds) == 0 {
		fmt.Println("No feeds currently being followed! Use follow <url> to follow a feed! OR addfeed <name> <url>!")
		return nil
	}
	for _, f := range feeds {
		fmt.Println(f)
	}
	return nil
}

type loggedInHandler func(s *state, cmd Command, user database.User) error

func middlewareLoggedIn(next loggedInHandler) func(*state, Command) error {
	return func(s *state, cmd Command) error {
		user, err := s.queries.GetUser(context.Background(), s.cfg.UserName)
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("current user not found, run login or register")
		}
		if err != nil {
			return err
		}
		return next(s, cmd, user)
	}
}

func handlerUnfollow(s *state, cmd Command, user database.User) error {
	if len(cmd.args) != 1 {
		return errors.New("usage: unfollow <url>")
	}
	deleteParams := database.DeleteFollowParams{
		UserID: user.ID,
		Url:    cmd.args[0],
	}
	err := s.queries.DeleteFollow(context.Background(), deleteParams)
	if err != nil {
		return err
	}
	fmt.Println(cmd.args[0] + " was unfollowed!")
	return nil
}

func scrapeFeeds(s *state) error {
	feed, err := s.queries.GetNextFeedToFetch(context.Background())
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("no feeds in database please add or follow")
	}
	if err != nil {
		return err
	}
	err = s.queries.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		return err
	}
	f, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}
	for _, item := range f.Channel.Item {
		var desc sql.NullString
		if item.Description != "" {
			desc = sql.NullString{String: item.Description, Valid: true}
		}
		t := time.Now()
		post := database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   t,
			UpdatedAt:   t,
			Title:       item.Title,
			Url:         item.Link,
			Description: desc,
			PublishedAt: parsePubDate(item.PubDate),
			FeedID:      feed.ID,
		}
		err = s.queries.CreatePost(context.Background(), post)
		if err != nil {
			fmt.Printf("couldn't save post %s: %v\n", item.Link, err)
		}
	}

	return nil
}

func parsePubDate(s string) sql.NullTime {
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		"Mon, 2 Jan 2006 15:04:05 -0700", // single-digit day
		"Mon, 2 Jan 2006 15:04:05 MST",   // single-digit day, named zone
	}
	for _, l := range layouts {
		t, err := time.Parse(l, s)
		if err == nil {
			return sql.NullTime{Time: t, Valid: true}
		}
	}
	return sql.NullTime{}
}

func handlerBrowse(s *state, cmd Command, user database.User) error {
	limit := 2
	if len(cmd.args) > 0 {
		var err error
		limit, err = strconv.Atoi(cmd.args[0])
		if err != nil {
			return err
		}
	}
	data := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	}
	posts, err := s.queries.GetPostsForUser(context.Background(), data)
	if err != nil {
		return err
	}
	if len(posts) < 1 {
		fmt.Println("Usage: Browse <limit>")
	}
	for _, p := range posts {
		fmt.Println(p.Title)
		fmt.Println(p.Url)
		fmt.Println(p.Description.String)
	}
	return nil
}
