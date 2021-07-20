package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/numary/ledger/api"
	"github.com/numary/ledger/config"
	"github.com/numary/ledger/ledger"
	"github.com/numary/ledger/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var (
	FlagBindAddr string
)

var root = &cobra.Command{
	Use: "numary",
}

func Execute() {
	server := &cobra.Command{
		Use: "server",
	}

	start := &cobra.Command{
		Use: "start",
		Run: func(cmd *cobra.Command, args []string) {
			app := fx.New(
				fx.Provide(
					ledger.NewResolver,
					api.NewHttpAPI,
				),
				fx.Invoke(func() {
					config.Init()
				}),
				fx.Invoke(func(lc fx.Lifecycle, h *api.HttpAPI) {
				}),
			)

			app.Run()
		},
	}

	server.AddCommand(start)

	conf := &cobra.Command{
		Use: "config",
	}

	conf.AddCommand(&cobra.Command{
		Use: "init",
		Run: func(cmd *cobra.Command, args []string) {
			config.Init()
			err := viper.SafeWriteConfig()
			if err != nil {
				fmt.Println(err)
			}
		},
	})

	store := &cobra.Command{
		Use: "storage",
	}

	store.AddCommand(&cobra.Command{
		Use: "init",
		Run: func(cmd *cobra.Command, args []string) {
			config.Init()
			s, err := storage.GetStore("default")

			if err != nil {
				log.Fatal(err)
			}

			err = s.Initialize()

			if err != nil {
				log.Fatal(err)
			}
		},
	})

	script := &cobra.Command{
		Use:  "exec [ledger] [script]",
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			config.Init()

			b, err := ioutil.ReadFile(args[1])

			if err != nil {
				log.Fatal(err)
			}

			r := regexp.MustCompile(`^\n`)
			s := string(b)
			s = r.ReplaceAllString(s, "")

			b, err = json.Marshal(gin.H{
				"plain": string(s),
			})

			if err != nil {
				log.Fatal(err)
			}

			res, err := http.Post(
				fmt.Sprintf(
					"http://%s/%s/script",
					viper.Get("server.http.bind_address"),
					args[0],
				),
				"application/json",
				bytes.NewReader([]byte(b)),
			)

			if err != nil {
				log.Fatal(err)
			}

			b, err = ioutil.ReadAll(res.Body)

			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(res.StatusCode, string(b))
		},
	}

	root.AddCommand(server)
	root.AddCommand(conf)
	root.AddCommand(UICmd)
	root.AddCommand(store)
	root.AddCommand(script)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
