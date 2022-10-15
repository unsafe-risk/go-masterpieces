package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"v8.run/go/broccoli"
)

type App struct {
	_    struct{} `version:"0.0.1" command:"uslibs" about:"Unsafe libraries"`
	DB   string   `flag:"db" default:"./list" about:"Database path" required:"true"`
	Add  *AddCmd  `subcommand:"add"`
	Del  *DelCmd  `subcommand:"del"`
	List *ListCmd `subcommand:"list"`
}

type AddCmd struct {
	_           struct{} `command:"add" about:"Add a library"`
	Name        string   `flag:"name" about:"Name of the library" required:"true"`
	URL         string   `flag:"url" about:"URL of the library" required:"true"`
	Description string   `flag:"description" about:"Description of the library"`
}

type DelCmd struct {
	_  struct{} `command:"del" about:"Delete a library"`
	ID uint64   `flag:"id" about:"ID of the library" required:"true"`
}

type ListCmd struct {
	_        struct{} `command:"list" about:"List libraries"`
	Limit    int      `flag:"limit" default:"-1" required:"true" about:"Limit the number of results"`
	JSON     bool     `flag:"json" about:"Output as JSON"`
	MARKDOWN bool     `flag:"markdown" about:"Output as Markdown"`
}

func PureLink(u string) string {
	uu, err := url.Parse(u)
	if err != nil {
		// Remove Prefix
		if strings.HasPrefix(u, "https://") {
			u = u[8:]
		} else if strings.HasPrefix(u, "http://") {
			u = u[7:]
		}
		return u
	}

	u = uu.Host + uu.Path
	if strings.HasSuffix(u, "/") {
		u = u[:len(u)-1]
	}
	return u
}

func main() {
	var app App
	_ = broccoli.BindOSArgs(&app)
	if app.DB == "" {
		app.DB = "./list"
	}
	initDB(app.DB)
	defer closeDB()
	switch {
	case app.Add != nil:
		err := AddLibrary(app.Add.Name, app.Add.URL, app.Add.Description)
		if err != nil {
			panic(err)
		}
	case app.List != nil:
		libs, err := ListLibraries(app.List.Limit)
		if err != nil {
			panic(err)
		}

		if app.List.JSON {
			err = json.NewEncoder(os.Stdout).Encode(libs)
			if err != nil {
				panic(err)
			}
		} else if app.List.MARKDOWN {
			w := bufio.NewWriter(os.Stdout)
			defer w.Flush()
			w.WriteString("# Library List\n\n")
			for _, lib := range libs {
				w.WriteString(fmt.Sprintf("## %s\n\n", lib.Name))
				w.WriteString(fmt.Sprintf("ID: %d\n\n", lib.ID))
				w.WriteString(fmt.Sprintf("URL: [%s](%s)\n\n", PureLink(lib.URL), lib.URL))
				w.WriteString(fmt.Sprintf("Description: %s\n\n", lib.Description))
			}
		}
	case app.Del != nil:
		err := DeleteLibrary(app.Del.ID)
		if err != nil {
			panic(err)
		}
	default:
		log.Println("App:", app)
		panic("unknown command")
	}
}
