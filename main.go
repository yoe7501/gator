package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/yoe7501/gator/internal/config"
	"github.com/yoe7501/gator/internal/database"
)

type state struct {
	cfg     *config.Config
	queries *database.Queries
}

func main() {
	c, err := config.Read()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	db, err := sql.Open("postgres", c.DBURL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	s := state{cfg: &c, queries: database.New(db)}
	cmds := Commands{
		commandToHandler: make(map[string]func(*state, Command) error),
	}
	cmds.register("login", handlerLogin)

	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", handlerAddFeed)
	if len(os.Args) < 2 {
		fmt.Println("usage: <command> <arguments>")
		os.Exit(1)
	}

	cmd := Command{
		name: os.Args[1],
		args: os.Args[2:],
	}
	err = cmds.run(&s, cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
