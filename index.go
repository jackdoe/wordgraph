package main

import (
	"archive/zip"
	"compress/gzip"
	"encoding/gob"
	metro "github.com/dgryski/go-metro"
	"github.com/montanaflynn/stats"
	"github.com/schollz/progressbar"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"unicode"
)

type Node struct {
	WordId     uint32
	WordString string
	Count      int
	Flow       map[uint64]*Edge
}

func (this *Node) addLinkThrough(from, to *Node) *Edge {
	key := uint64(from.WordId)<<32 | uint64(to.WordId)
	edge, ok := this.Flow[key]
	if !ok {
		edge = &Edge{
			Count:      0,
			FromNodeId: from.WordId,
			ToNodeId:   to.WordId,
		}
		this.Flow[key] = edge
	}

	return edge
}

type Edge struct {
	Count      int
	FromNodeId uint32
	ToNodeId   uint32
}

type Graph struct {
	Nodes map[uint32]*Node
}

type Index struct {
	Graph *Graph
	Words map[string]uint32
}

func NewIndex() *Index {
	return &Index{
		Graph: &Graph{
			Nodes: map[uint32]*Node{},
		},
		Words: map[string]uint32{},
	}
}

type tokenized struct {
	words []string
}

func tokenize(s string) []string {
	s = strings.Replace(s, ",", " , ", -1)
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && c != ','
	}

	return strings.FieldsFunc(strings.ToLower(s), f)
}

func hash(s string) uint64 {
	return metro.Hash64Str(s, 0)
}

func (this *Index) info() int {
	fromN := 0
	for _, node := range this.Graph.Nodes {
		fromN += len(node.Flow)
	}
	return fromN
}

func (this *Index) save(fn string) {
	log.Printf("saving to %s", fn)
	f, err := os.Create(fn)

	writer, _ := gzip.NewWriterLevel(f, gzip.BestCompression)

	if err != nil {
		log.Fatal(err)
	}
	enc := gob.NewEncoder(writer)
	err = enc.Encode(this)
	if err != nil {
		log.Fatal(err)
	}
	writer.Close()
	f.Close()
}

func (this *Index) load(fn string) {
	log.Printf("loading from %s", fn)
	f, err := os.Open(fn)
	reader, err := gzip.NewReader(f)

	if err != nil {
		log.Fatal(err)
	}
	dec := gob.NewDecoder(reader)
	err = dec.Decode(this)

	if err != nil {
		log.Fatal(err)
	}
	f.Close()

}

func (this *Index) parse(root string) {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		log.Fatal(err)
	}
	toParse := []string{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".txt") || strings.HasSuffix(file.Name(), ".zip") {
			toParse = append(toParse, path.Join(root, file.Name()))
		}
	}

	wordcount := map[uint64]uint32{}
	maxWordThresh := uint32(15)

	createWordCount := func(done tokenized) {
		for _, w := range done.words {
			wordcount[hash(w)]++
		}
	}

	createNodes := func(done tokenized) {
		var prevNode *Node
		var prevPrevNode *Node

		for _, w := range done.words {
			wc := wordcount[hash(w)]
			if wc < maxWordThresh {
				continue
			}

			wordId, ok := this.Words[w]
			if !ok {
				wordId = uint32(len(this.Words))
				this.Words[w] = wordId
			}

			node, ok := this.Graph.Nodes[wordId]

			if !ok {
				node = &Node{
					Flow:       map[uint64]*Edge{},
					WordId:     wordId,
					WordString: w,
					Count:      0,
				}
				this.Graph.Nodes[wordId] = node
			}

			node.Count++
			if prevNode != nil && prevPrevNode != nil {
				edge := prevNode.addLinkThrough(prevPrevNode, node)
				edge.Count++
			}
			prevPrevNode = prevNode
			prevNode = node
		}
	}

	tokenizeFiles := func(info string, callbackPerFile func(tokenized)) {
		chunkSize := 10
		log.Printf("\n[%s]\n", info)
		bar := progressbar.New(len(toParse))
		for i := 0; i < len(toParse); i += chunkSize {
			end := i + chunkSize
			if end > len(toParse) {
				end = len(toParse)
			}
			doneChannel := make(chan tokenized, chunkSize)
			maxFileSize := uint64(1000000)
			chunk := toParse[i:end]
			for _, x := range chunk {
				go func(file string) {
					var dat []byte
					fi, err := os.Stat(file)
					if err != nil {
						log.Printf("%s %s", file, err.Error())
						doneChannel <- tokenized{[]string{}}
						return
					}
					if uint64(fi.Size()) > maxFileSize {
						doneChannel <- tokenized{[]string{}}
						return
					}

					if strings.HasSuffix(file, ".zip") {
						r, err := zip.OpenReader(file)
						if err != nil {
							log.Printf("%s %s", file, err.Error())
							doneChannel <- tokenized{[]string{}}
							return
						}

						for _, f := range r.File {
							if f.UncompressedSize64 > maxFileSize {
								doneChannel <- tokenized{[]string{}}
								return
							}

							if strings.HasSuffix(f.Name, ".txt") {
								rc, err := f.Open()
								if err != nil {
									log.Printf("%s %s", f.Name, err.Error())
									rc.Close()
									continue
								}
								dat, err = ioutil.ReadAll(rc)
								if err != nil {
									log.Printf("%s %s", f.Name, err.Error())
									rc.Close()
									continue
								}
								rc.Close()
							}
						}
						r.Close()
					} else {
						dat, err = ioutil.ReadFile(file)
						if err != nil {
							log.Fatal(err)
						}

					}

					words := tokenize(string(dat))
					doneChannel <- tokenized{words}
				}(x)
			}

			for i := 0; i < len(chunk); i++ {
				done := <-doneChannel
				callbackPerFile(done)
				bar.Add(1)
			}
			close(doneChannel)
		}
	}

	tokenizeFiles("word count", func(done tokenized) {
		createWordCount(done)
	})

	i := 0
	tokenizeFiles("create graph", func(done tokenized) {
		createNodes(done)

		i++
		if i > 1000 {
			// XXX: i dont have enough ram to do this after the whole thing is built
			before := this.info()
			for _, node := range this.Graph.Nodes {
				threshFrom := getThresh(node.Flow, 75)
				for id, edge := range node.Flow {
					if float64(edge.Count) < threshFrom {
						delete(node.Flow, id)
					}
				}
			}
			after := this.info()
			log.Printf("\ncleaning up the graph nodes: %d, edges before: %d, after: %d\n", len(this.Graph.Nodes), before, after)
			runtime.GC()
			i = 0
		}
	})
}

type Flow struct {
	Words []string
	Count int
}

type WordItem struct {
	Flows []*Flow
	Word  string
}

type QueryResult struct {
	Tokenized []string
	Processed map[string]*WordItem
	Error     string
}

func getThresh(edges map[uint64]*Edge, per float64) float64 {
	f := make([]float64, len(edges))
	i := 0
	for _, edge := range edges {
		f[i] = float64(edge.Count)
		i++
	}

	percentile, _ := stats.Percentile(f, per)
	return 1 + percentile
}

func (this *Index) query(text string, percentile float64) *QueryResult {
	wtext := tokenize(text)
	if len(wtext) > 30 {
		return &QueryResult{
			Error: "too many words, please ask for at most 30 words at a time",
		}
	}
	out := &QueryResult{
		Tokenized: wtext,
		Processed: map[string]*WordItem{},
	}
	if percentile == 0 {
		percentile = 95
	}
	if percentile <= 75 {
		percentile = 75
	}
	wordMap := map[uint32]bool{}

	for _, w := range wtext {
		wordId, ok := this.Words[w]
		if !ok {
			continue
		}
		wordMap[wordId] = true
	}

	for _, w := range wtext {
		if w == "," {
			continue
		}
		wordItem, ok := out.Processed[w]
		if !ok {
			wordId, ok := this.Words[w]
			if !ok {
				continue
			}

			node, ok := this.Graph.Nodes[wordId]
			if !ok {
				continue
			}

			wordItem = &WordItem{
				Flows: []*Flow{},
				Word:  w,
			}

			threshFrom := getThresh(node.Flow, percentile)

			for _, edge := range node.Flow {
				inFlow := wordMap[edge.FromNodeId] || wordMap[edge.ToNodeId]
				if float64(edge.Count) > threshFrom || inFlow {
					if edge.FromNodeId != node.WordId && edge.FromNodeId != edge.ToNodeId {
						a := this.Graph.Nodes[edge.FromNodeId].WordString
						b := node.WordString
						c := this.Graph.Nodes[edge.ToNodeId].WordString
						wordItem.Flows = append(wordItem.Flows, &Flow{
							Words: []string{a, b, c},
							Count: edge.Count,
						})
					}
				}
			}

			sort.Slice(wordItem.Flows, func(i, j int) bool {
				return wordItem.Flows[j].Count < wordItem.Flows[i].Count
			})
			out.Processed[w] = wordItem
		}
	}
	return out
}

type FlowResultItem struct {
	Word  string
	Score float64
}

type FlowResult struct {
	Items []*FlowResultItem
}

func (this *Index) flow(text string) *FlowResult {
	wtext := tokenize(text)
	out := &FlowResult{
		Items: make([]*FlowResultItem, len(wtext)),
	}
	var prevNode *Node
	var prevPrevNode *Node
	for i := 0; i < len(wtext); i++ {
		out.Items[i] = &FlowResultItem{
			Word:  wtext[i],
			Score: 0,
		}
	}
	explainAndScore := func(f *FlowResultItem, s string, score float64) {
		//		f.Explain = append(f.Explain, s)
		f.Score += score
	}

	for i := 0; i < len(wtext); i++ {
		w := wtext[i]
		wordId, ok := this.Words[w]
		if !ok {
			continue
		}

		node, ok := this.Graph.Nodes[wordId]
		if !ok {
			continue
		}

		if prevNode != nil && prevPrevNode == nil {
			// XXX: not sure about this
			for key, _ := range prevNode.Flow {
				if (key & uint64(0xFFFFFFFF)) == uint64(node.WordId) {
					explainAndScore(out.Items[i-1], node.WordString, 1)
					explainAndScore(out.Items[i], node.WordString, 2)
					break
				}
			}
		}

		// prevprevnode -> node through prevNode

		if prevPrevNode != nil {
			key := uint64(prevPrevNode.WordId)<<32 | uint64(node.WordId)
			_, ok := prevNode.Flow[key]
			if ok {
				explainAndScore(out.Items[i-2], node.WordString, 1)
				explainAndScore(out.Items[i-1], node.WordString, 1)
				explainAndScore(out.Items[i], node.WordString, 3)
			}
		}

		prevPrevNode = prevNode
		prevNode = node
	}
	return out
}
