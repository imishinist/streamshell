package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// {"result-set":{"docs":[
//    {"from":"1800flowers.133139412@s2u2.com","to":"lcampbel@enron.com"},
//    {"from":"1800flowers.93690065@s2u2.com","to":"jtholt@ect.enron.com"},
//    {"from":"1800flowers.96749439@s2u2.com","to":"alewis@enron.com"},
//    {"from":"1800flowers@1800flowers.flonetwork.com","to":"lcampbel@enron.com"},
//    {"from":"1800flowers@shop2u.com","to":"lcampbel@enron.com"},
//    {"from":"1800flowers@shop2u.com","to":"ebass@enron.com"},
//    {"EOF":true,"RESPONSE_TIME":33}]}
// }
func decodeStreamJson(ctx context.Context, body io.ReadCloser, depth int) <-chan json.RawMessage {
	ret := make(chan json.RawMessage)
	dec := json.NewDecoder(body)

	go func() {
		defer body.Close()
		defer close(ret)

		delimCounts := 0
		for i := 0; ; i++ {
			t, err := dec.Token()
			if err != nil {
				log.Println(err)
				return
			}
			if _, ok := t.(json.Delim); ok {
				delimCounts++
			}

			if delimCounts >= depth {
				break
			}
		}

		for dec.More() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var doc json.RawMessage
			// decode an array value (Message)
			err := dec.Decode(&doc)
			if err != nil {
				log.Println(err)
				return
			}

			ret <- doc
		}

		for i := 0; i < delimCounts; i++ {
			_, err := dec.Token()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()

	return ret
}

type Endpoint struct {
	Host string
	Port int

	Collection string
}

func (e *Endpoint) StreamURL() string {
	return fmt.Sprintf("http://%s:%d/solr/%s/stream", e.Host, e.Port, e.Collection)
}

type SolrClient struct {
	client *http.Client
}

func (s *SolrClient) Stream(ctx context.Context, endpoint Endpoint, body string) (<-chan json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.StreamURL(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	docs := decodeStreamJson(ctx, res.Body, 3)
	return docs, nil
}

type Prompt struct {
	Input  io.Reader
	Output io.Writer
}

func (p *Prompt) Show() {
	fmt.Fprint(p.Output, "> ")
}

func (p *Prompt) GetLine() string {
	var line string
	fmt.Fscan(p.Input, &line)
	return line
}

func main() {
	host := flag.String("host", "localhost", "solr host")
	port := flag.Int("port", 8983, "solr port")
	collection := flag.String("collection", "techproducts", "solr collection")
	interactive := flag.Bool("i", false, "interactive mode")
	flag.Parse()

	client := &SolrClient{
		client: http.DefaultClient,
	}
	endpoint := Endpoint{
		Host:       *host,
		Port:       *port,
		Collection: *collection,
	}
	prompt := &Prompt{
		Input:  os.Stdin,
		Output: os.Stdout,
	}

	if !*interactive {
		expr := "expr=" + prompt.GetLine()
		ctx := context.Background()
		docs, err := client.Stream(ctx, endpoint, expr)
		if err != nil {
			log.Fatal(err)
		}
		for doc := range docs {
			fmt.Println(string(doc))
		}
		return
	}

	for {
		prompt.Show()
		line := prompt.GetLine()
		if strings.TrimSpace(line) == "" {
			continue
		}

		expr := "expr=" + line

		ctx := context.Background()
		docs, err := client.Stream(ctx, endpoint, expr)
		if err != nil {
			fmt.Println(err)
			continue
		}
		for doc := range docs {
			fmt.Println(string(doc))
		}
	}
}
