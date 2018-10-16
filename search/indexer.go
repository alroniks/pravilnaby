package main

import (
    "fmt"
    "sync"
    "io/ioutil"
    "strconv"
    "strings"
    "regexp"
    "sort"
    "encoding/json"
    "github.com/microcosm-cc/bluemonday"
    "github.com/russross/blackfriday"
)

type Meta struct {
    Title string
    Chapter string
    Section string
}

type Content struct {
    Point int
    Text string
    Combinations []string
    Examples []string
}

type Entry struct {

    ID int `json:"objectID"`
    Section string `json:"section"`
    Chapter string `json:"chapter"`
    Title string `json:"title"`
    Paragraph int `json:"paragraph"`
    Point int `json:"point"`
    Combinations []string `json:"combinations"`
    Examples []string `json:"examples"`
    Text string `json:"text"`
    Link string `json:"link"`

}

var wg sync.WaitGroup
var tube chan Entry

func main() {

    tube = make(chan Entry)

    counter := 1;

    for counter <= 66 {
        wg.Add(1)
        go processParagraph(counter)
        counter++;
    }

    go func() {
        wg.Wait()
        close(tube)
    }()


    var out []Entry = []Entry{}
    for entry := range tube {
        out = append(out, entry)
    }

    sort.Slice(out, func (i, j int) bool {
        return out[i].ID < out[j].ID
    })

    json, _ := json.MarshalIndent(out, "", "  ")

    fmt.Println(string(json))
}

func processParagraph(p int) {
    defer wg.Done()

    path := fmt.Sprintf("../content/rules/%d.md", p)

    bytes, err := ioutil.ReadFile(path)
    
    if err != nil {
        return
    }

    meta, content := split(string(bytes), "+++")

    metadata := handleMetaData(meta)

    for _, data := range handleContent(content) {

        number := fmt.Sprintf("%d%d", p, data.Point)
        id, _ := strconv.ParseInt(number, 10, 32)

        title := fmt.Sprintf("§%d. %s, #%d", p, metadata.Title, data.Point)

        link := fmt.Sprintf("https://pravilna.by/rules/%d/#p%d", p, data.Point)

        tube <- Entry{
            ID: int(id),
            Section: metadata.Section,
            Chapter: metadata.Chapter,
            Title: title,
            Paragraph: p,
            Point: data.Point,
            Combinations: data.Combinations,
            Examples: data.Examples,
            Text: strings.Replace(data.Text, "\n", "", -1),
            Link: link,
        }   
    }
}

func handleMetaData(input string) Meta {

    metadata := Meta{}

    rows := strings.Split(input, "\n")

    for _, row := range rows {
        parts := strings.Split(row, " = ")

        switch f := parts[0]; f {
            case "title":
                metadata.Title = strings.Trim(parts[1], "\"")
            case "chapter":
                metadata.Chapter = strings.Trim(parts[1], "\"")
            case "section":
                metadata.Section = strings.Trim(parts[1], "\"")
        }
    }

    return metadata
}

func findEntities(input string, rule string) []string {

    exp := regexp.MustCompile(rule)

    var out []string = []string{}

    for _, match := range exp.FindAllStringSubmatch(input, -1) {
        out = append(out, match[1])
    }

    return out
}

func getPointNumber(input string) (int, string) {
    exp := regexp.MustCompile(`^\d\.\s`)

    number := strings.Replace(exp.FindString(input), ". ", "", -1)
    rest := exp.ReplaceAllString(input, "")

    p, _ := strconv.ParseInt(number, 10, 32)

    return int(p), rest
}

func handleContent(input string) []Content {

    bm := bluemonday.StrictPolicy()
    prepare := bm.Sanitize(input)

    prepare = strings.Replace(prepare, "\n\n", "---", -1)
    prepare = strings.Replace(prepare, "\n", " ", -1)
    prepare = strings.Replace(prepare, "---", "\n\n", -1)

    exp := regexp.MustCompile(`(?m)^\d\.\s.+$`)

    var out []Content = []Content{}

    for _, match := range exp.FindAllString(prepare, -1) {

        point, text := getPointNumber(match)

        parsed := blackfriday.MarkdownCommon([]byte(text))

        combinations := findEntities(string(parsed), "<strong>(.*?)</strong>")
        examples := findEntities(string(parsed), "<em>(.*?)</em>")

        out = append(out, Content{
            Point: point,
            Text: bm.Sanitize(string(parsed)),
            Combinations: combinations,
            Examples: examples,
        })
    }

    return out;
}

func split(source string, delimiter string) (string, string) {
    out := strings.Split(strings.TrimLeft(source, delimiter), delimiter)

    metadata := strings.TrimSpace(out[0])
    content := strings.TrimSpace(out[1])
    
    return metadata, content
}