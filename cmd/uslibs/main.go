package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"

	"v8.run/go/broccoli"
)

type App struct {
	_    struct{}   `version:"0.0.1" command:"uslibs" about:"Unsafe libraries"`
	DB   string     `flag:"db" default:"./list" about:"Database path" required:"true"`
	Add  *AddCmd    `subcommand:"add"`
	Del  *DelCmd    `subcommand:"del"`
	List *ListCmd   `subcommand:"list"`
	Back *BackupCmd `subcommand:"backup"`
}

type AddCmd struct {
	_           struct{} `command:"add" about:"Add a library"`
	Name        string   `flag:"name" about:"Name of the library" required:"true"`
	URL         string   `flag:"url" about:"URL of the library" required:"true"`
	Description string   `flag:"description" about:"Description of the library"`
	Tags        string   `flag:"tags" about:"Tags of the library"`
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

type BackupCmd struct {
	_ struct{} `command:"backup" about:"Backup the database"`
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

func FormatName(lib Library) string {
	return fmt.Sprintf("0x%02X. %s", lib.ID, lib.Name)
}

var Links = map[string]string{}

func GetLink(l string) string {
	if v, ok := Links[l]; ok {
		return v
	}
	al := GetAnchorLink(l)
	Links[l] = al
	return al
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
		tags := strings.Split(app.Add.Tags, ",")
		err := AddLibrary(app.Add.Name, app.Add.URL, app.Add.Description, tags...)
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
			w.WriteString("# Go Masterpieces\n\n")
			GetLink("Go Masterpieces")
			w.WriteString("Masterpieces of Go programming language.\n\n")

			var allTagsMap = map[string][]Library{}
			// Write Tags
			w.WriteString("# Tags\n\n")
			GetLink("Tags")
			for _, lib := range libs {
				for _, tag := range lib.Tags {
					allTagsMap[tag] = append(allTagsMap[tag], lib)
				}
			}

			var allTags []string
			for tag := range allTagsMap {
				allTags = append(allTags, tag)
			}
			sort.Strings(allTags)
			for _, tag := range allTags {
				w.WriteString(fmt.Sprintf("## %s\n\n", tag))
				GetLink(tag)
				for _, lib := range allTagsMap[tag] {
					w.WriteString(fmt.Sprintf("* [%s](%s)\n", FormatName(lib), GetLink(FormatName(lib))))
				}
				w.WriteString("\n\n")
			}
			w.WriteString("\n\n")

			w.WriteString("# Masterpieces\n\n")
			sort.Slice(libs, func(i, j int) bool {
				return libs[i].ID < libs[j].ID
			})
			for _, lib := range libs {
				w.WriteString(fmt.Sprintf("## %s\n\n", FormatName(lib)))
				w.WriteString(fmt.Sprintf("URL: [%s](%s)\n\n", PureLink(lib.URL), lib.URL))
				// Write Tags
				if len(lib.Tags) > 0 {
					w.WriteString("Tags: ")
					for i, tag := range lib.Tags {
						if i > 0 {
							w.WriteString(", ")
						}
						w.WriteString(fmt.Sprintf("[%s](%s)", tag, GetLink(tag)))
					}
					w.WriteString("\n\n")
				}
				w.WriteString(fmt.Sprintf("%s\n\n", lib.Description))

				w.WriteString("\n\n")
			}

		}
	case app.Del != nil:
		err := DeleteLibrary(app.Del.ID)
		if err != nil {
			panic(err)
		}
	case app.Back != nil:
		err := BackupDB()
		if err != nil {
			panic(err)
		}
	default:
		log.Println("App:", app)
		panic("unknown command")
	}
}
