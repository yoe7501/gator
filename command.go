package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	feed, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}
	fmt.Println(feed)
	return nil
}

func handlerAddFeed(s *state, cmd Command) error {
	if len(cmd.args) != 2 {
		return errors.New("addfeed <name> <url>")
	}
	user, err := s.queries.GetUser(context.Background(), s.cfg.UserName)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("current user not found, run login or register")
	}
	if err != nil {
		return err
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

func handlerFollow(s *state, cmd Command) error {
	if len(cmd.args) != 1 {
		return errors.New("follow <url>")
	}
	user, err := s.queries.GetUser(context.Background(), s.cfg.UserName)
	if err != nil {
		return err
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

func handlerFollowing(s *state, cmd Command) error {
	user, err := s.queries.GetUser(context.Background(), s.cfg.UserName)
	if err != nil {
		return err
	}
	feeds, err := s.queries.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}
	if len(feeds) == 0 {
		fmt.Println("No feeds currently being followed! Use follow <url> to follow a feed! OR addfeed <name> <url>!")
	}
	for _, f := range feeds {
		fmt.Println(f)
	}
	return nil
}
